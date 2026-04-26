package assistant

import (
	"aiguide/internal/pkg/constant"
	"aiguide/internal/pkg/tools"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	adkmodel "google.golang.org/adk/model"
	adksession "google.golang.org/adk/session"
	"google.golang.org/genai"
)

const defaultLiveModel = "gemini-3.1-flash-live-preview"

const (
	phaseWaiting = "waiting"
	phaseUser    = "user"
	phaseModel   = "model"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type turnTracker struct {
	phase      string
	userAudio  [][]byte
	userText   string
	modelAudio [][]byte
	modelText  string

	mu           sync.Mutex
	pendingAudio [][]byte
	pendingText  string
	pendingTimer *time.Timer
}

func (a *Assistant) VoiceCall(ctx *gin.Context) {
	userID, ok := getContextUserID(ctx)
	if !ok || userID <= 0 {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessionID := ctx.Query("session_id")
	if sessionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	conn, err := wsUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		slog.Error("VoiceCall: WebSocket upgrade failed", "err", err, "userID", userID)
		return
	}
	defer conn.Close()

	slog.Info("VoiceCall: WebSocket connected", "userID", userID, "sessionID", sessionID)

	userIDStr := strconv.Itoa(userID)
	if _, err := a.ensureSession(ctx, userIDStr, sessionID); err != nil {
		slog.Error("VoiceCall: ensureSession failed", "err", err)
		writeWSError(conn, "failed to ensure session: "+err.Error())
		return
	}

	connCtx, connCancel := context.WithCancel(ctx.Request.Context())
	defer connCancel()

	liveSession, err := a.connectLiveSession(connCtx, userID)
	if err != nil {
		writeWSError(conn, err.Error())
		return
	}
	defer liveSession.Close()

	writeCh := make(chan wsServerMessage, 32)
	defer close(writeCh)
	go wsWriter(conn, writeCh, connCancel)

	writeCh <- wsServerMessage{Type: serverMsgTypeSetupOK}

	saveTurn, waitSaves := a.startSaveLoop(userID, sessionID)
	defer waitSaves()

	firstErr := a.bridgeVoiceCall(connCtx, connCancel, writeCh, liveSession, conn, saveTurn)

	if firstErr != nil {
		slog.Info("VoiceCall: session ended", "reason", firstErr.Error(), "userID", userID)
	} else {
		slog.Info("VoiceCall: session ended normally", "userID", userID)
	}
}

func (a *Assistant) connectLiveSession(ctx context.Context, userID int) (*genai.Session, error) {
	liveClient, err := a.getLiveClient(ctx)
	if err != nil {
		slog.Error("VoiceCall: getLiveClient failed", "err", err, "userID", userID)
		return nil, fmt.Errorf("failed to initialize live client: %w", err)
	}

	liveModel := a.liveModel
	if liveModel == "" {
		liveModel = defaultLiveModel
	}

	session, err := liveClient.Live.Connect(ctx, liveModel, &genai.LiveConnectConfig{
		ResponseModalities: []genai.Modality{genai.ModalityAudio},
		SpeechConfig: &genai.SpeechConfig{
			VoiceConfig: &genai.VoiceConfig{
				PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{
					VoiceName: "Kore",
				},
			},
		},
		InputAudioTranscription:  &genai.AudioTranscriptionConfig{},
		OutputAudioTranscription: &genai.AudioTranscriptionConfig{},
	})
	if err != nil {
		slog.Error("VoiceCall: Live.Connect failed", "err", err, "userID", userID)
		return nil, fmt.Errorf("failed to connect to Gemini Live: %w", err)
	}
	return session, nil
}

func wsWriter(conn *websocket.Conn, writeCh <-chan wsServerMessage, cancel context.CancelFunc) {
	for msg := range writeCh {
		data, err := json.Marshal(msg)
		if err != nil {
			slog.Error("VoiceCall: json.Marshal error", "err", err)
			cancel()
			return
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			cancel()
			return
		}
	}
}

type saveRequest struct {
	role       string
	text       string
	audio      [][]byte
	sampleRate uint32
}

func (a *Assistant) startSaveLoop(userID int, sessionID string) (saveTurn func(string, string, [][]byte, uint32), wait func()) {
	saveCh := make(chan saveRequest, 16)
	saveDone := make(chan struct{})
	go func() {
		defer close(saveDone)
		for req := range saveCh {
			a.saveVoiceTurn(context.Background(), userID, sessionID, req.role, req.text, req.audio, req.sampleRate)
		}
	}()

	saveTurn = func(role, text string, audioChunks [][]byte, sampleRate uint32) {
		slog.Info("saveTurn: queued", "role", role, "textLen", len(text), "audioChunks", len(audioChunks))
		copied := make([][]byte, len(audioChunks))
		copy(copied, audioChunks)
		saveCh <- saveRequest{role: role, text: text, audio: copied, sampleRate: sampleRate}
	}

	wait = func() {
		close(saveCh)
		<-saveDone
	}
	return
}

func (a *Assistant) bridgeVoiceCall(
	ctx context.Context,
	cancel context.CancelFunc,
	writeCh chan<- wsServerMessage,
	liveSession *genai.Session,
	conn *websocket.Conn,
	saveTurn func(string, string, [][]byte, uint32),
) error {
	tracker := &turnTracker{phase: phaseWaiting}

	errCh := make(chan error, 2)
	go func() {
		errCh <- a.receiveFromGeminiWithTracking(writeCh, liveSession, tracker, saveTurn)
	}()
	go func() {
		errCh <- receiveFromClientWithTracking(conn, liveSession, cancel, tracker)
	}()

	firstErr := <-errCh
	cancel()

	tracker.flushPending(saveTurn)
	if tracker.userText != "" || len(tracker.userAudio) > 0 {
		saveTurn("user", tracker.userText, tracker.userAudio, 16000)
	}
	if tracker.phase == phaseModel && (tracker.modelText != "" || len(tracker.modelAudio) > 0) {
		saveTurn("model", tracker.modelText, tracker.modelAudio, 24000)
	}

	return firstErr
}

func (a *Assistant) receiveFromGeminiWithTracking(writeCh chan<- wsServerMessage, liveSession *genai.Session, tracker *turnTracker, saveTurn func(string, string, [][]byte, uint32)) error {
	for {
		msg, err := liveSession.Receive()
		if err != nil {
			tracker.flushPending(saveTurn)
			return fmt.Errorf("session.Receive: %w", err)
		}

		if msg.GoAway != nil {
			writeCh <- wsServerMessage{Type: serverMsgTypeGoAway}
		}

		sc := msg.ServerContent
		if sc == nil {
			continue
		}

		if sc.ModelTurn != nil {
			tracker.handleModelTurn(sc.ModelTurn.Parts, saveTurn, writeCh)
		}
		if sc.InputTranscription != nil && sc.InputTranscription.Text != "" {
			tracker.handleInputTranscription(sc.InputTranscription, saveTurn, writeCh)
		}
		if sc.OutputTranscription != nil && sc.OutputTranscription.Text != "" {
			tracker.handleOutputTranscription(sc.OutputTranscription, saveTurn, writeCh)
		}
		if sc.Interrupted {
			tracker.handleInterrupted(saveTurn, writeCh)
		}
		if sc.TurnComplete {
			tracker.handleTurnComplete(saveTurn, writeCh)
		}
	}
}

func (t *turnTracker) flushPending(saveTurn func(string, string, [][]byte, uint32)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pendingTimer != nil {
		t.pendingTimer.Stop()
		t.pendingTimer = nil
	}
	if len(t.pendingAudio) > 0 || t.pendingText != "" {
		saveTurn("model", t.pendingText, t.pendingAudio, 24000)
		t.pendingAudio = nil
		t.pendingText = ""
	}
}

func (t *turnTracker) handleModelTurn(parts []*genai.Part, saveTurn func(string, string, [][]byte, uint32), writeCh chan<- wsServerMessage) {
	for _, part := range parts {
		if part.InlineData == nil || len(part.InlineData.Data) == 0 {
			continue
		}
		if t.phase != phaseModel {
			if t.userText != "" || len(t.userAudio) > 0 {
				saveTurn("user", t.userText, t.userAudio, 16000)
			}
			t.phase = phaseModel
			t.userAudio = t.userAudio[:0]
			t.userText = ""
			t.modelAudio = t.modelAudio[:0]
			t.modelText = ""
		}
		t.modelAudio = append(t.modelAudio, part.InlineData.Data)

		writeCh <- wsServerMessage{
			Type: serverMsgTypeAudio,
			Data: base64.StdEncoding.EncodeToString(part.InlineData.Data),
		}
	}
}

func (t *turnTracker) handleInputTranscription(tr *genai.Transcription, saveTurn func(string, string, [][]byte, uint32), writeCh chan<- wsServerMessage) {
	t.flushPending(saveTurn)
	if t.phase != phaseUser {
		if t.phase == phaseModel && (t.modelText != "" || len(t.modelAudio) > 0) {
			saveTurn("model", t.modelText, t.modelAudio, 24000)
		}
		t.phase = phaseUser
		t.userText = ""
	}
	t.userText += tr.Text

	writeCh <- wsServerMessage{
		Type:     serverMsgTypeInputTranscript,
		Data:     tr.Text,
		Finished: tr.Finished,
	}
}

func (t *turnTracker) handleOutputTranscription(tr *genai.Transcription, saveTurn func(string, string, [][]byte, uint32), writeCh chan<- wsServerMessage) {
	t.mu.Lock()
	hasPending := t.pendingAudio != nil
	if hasPending {
		t.pendingText += tr.Text
	} else {
		t.modelText += tr.Text
	}
	t.mu.Unlock()

	writeCh <- wsServerMessage{
		Type:     serverMsgTypeOutputTranscript,
		Data:     tr.Text,
		Finished: tr.Finished,
	}

	if tr.Finished && hasPending {
		t.flushPending(saveTurn)
	}
}

func (t *turnTracker) handleInterrupted(saveTurn func(string, string, [][]byte, uint32), writeCh chan<- wsServerMessage) {
	t.flushPending(saveTurn)
	if t.phase == phaseModel && (t.modelText != "" || len(t.modelAudio) > 0) {
		saveTurn("model", t.modelText, t.modelAudio, 24000)
	}
	t.phase = phaseWaiting
	writeCh <- wsServerMessage{Type: serverMsgTypeInterrupted}
}

func (t *turnTracker) handleTurnComplete(saveTurn func(string, string, [][]byte, uint32), writeCh chan<- wsServerMessage) {
	slog.Info("handleTurnComplete", "phase", t.phase, "modelTextLen", len(t.modelText), "modelAudioChunks", len(t.modelAudio))
	if t.phase == phaseModel && (t.modelText != "" || len(t.modelAudio) > 0) {
		if t.modelText != "" {
			saveTurn("model", t.modelText, t.modelAudio, 24000)
		} else {
			slog.Info("handleTurnComplete: deferring save (no transcript yet)")
			t.mu.Lock()
			t.pendingAudio = make([][]byte, len(t.modelAudio))
			copy(t.pendingAudio, t.modelAudio)
			t.pendingText = ""
			t.pendingTimer = time.AfterFunc(3*time.Second, func() {
				t.mu.Lock()
				audio := t.pendingAudio
				text := t.pendingText
				t.pendingAudio = nil
				t.pendingText = ""
				t.pendingTimer = nil
				t.mu.Unlock()
				if len(audio) > 0 || text != "" {
					saveTurn("model", text, audio, 24000)
				}
			})
			t.mu.Unlock()
		}
	}
	t.phase = phaseWaiting
	writeCh <- wsServerMessage{Type: serverMsgTypeTurnComplete}
}

func receiveFromClientWithTracking(conn *websocket.Conn, liveSession *genai.Session, cancel context.CancelFunc, tracker *turnTracker) error {
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			cancel()
			return nil
		}

		var msg wsClientMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			slog.Warn("receiveFromClient: invalid JSON", "err", err)
			continue
		}

		switch msg.Type {
		case clientMsgTypeAudio:
			pcm, err := base64.StdEncoding.DecodeString(msg.Data)
			if err != nil {
				slog.Warn("receiveFromClient: base64 decode error", "err", err)
				continue
			}
			tracker.userAudio = append(tracker.userAudio, pcm)

			if err := liveSession.SendRealtimeInput(genai.LiveRealtimeInput{
				Audio: &genai.Blob{
					Data:     pcm,
					MIMEType: "audio/pcm;rate=16000",
				},
			}); err != nil {
				return fmt.Errorf("SendRealtimeInput: %w", err)
			}

		case clientMsgTypeEnd:
			if err := liveSession.SendRealtimeInput(genai.LiveRealtimeInput{
				AudioStreamEnd: true,
			}); err != nil {
				return fmt.Errorf("SendRealtimeInput(end): %w", err)
			}

		case clientMsgTypeText:
			if msg.Data == "" {
				continue
			}
			if err := liveSession.SendClientContent(genai.LiveClientContentInput{
				Turns: []*genai.Content{
					genai.NewContentFromText(msg.Data, genai.RoleUser),
				},
			}); err != nil {
				return fmt.Errorf("SendClientContent: %w", err)
			}
		}
	}
}

func (a *Assistant) saveVoiceTurn(ctx context.Context, userID int, sessionID, role, transcript string, audioChunks [][]byte, sampleRate uint32) {
	if transcript == "" && len(audioChunks) == 0 {
		slog.Info("saveVoiceTurn: skipped (empty)", "role", role, "sessionID", sessionID)
		return
	}

	slog.Info("saveVoiceTurn: starting", "role", role, "transcriptLen", len(transcript), "audioChunks", len(audioChunks), "sessionID", sessionID)

	fileID := a.saveVoiceAudio(ctx, userID, sessionID, role, audioChunks, sampleRate)

	metadataJSON, _ := json.Marshal(voiceAudioMetadata{FileID: fileID})
	messageText := fmt.Sprintf("%s %s -->\n%s", voiceAudioMetadataPrefix, string(metadataJSON), transcript)

	contentRole := genai.RoleUser
	author := "user"
	if role == "model" {
		contentRole = "model"
		author = "assistant"
	}

	userIDStr := strconv.Itoa(userID)
	getResp, err := a.session.Get(ctx, &adksession.GetRequest{
		AppName:   constant.AppNameAssistant.String(),
		UserID:    userIDStr,
		SessionID: sessionID,
	})
	if err != nil {
		slog.Error("saveVoiceTurn: session.Get() error", "err", err)
		return
	}

	event := &adksession.Event{
		ID:        uuid.NewString(),
		Timestamp: time.Now(),
		Author:    author,
		LLMResponse: adkmodel.LLMResponse{
			Content: &genai.Content{
				Role:  contentRole,
				Parts: []*genai.Part{genai.NewPartFromText(messageText)},
			},
		},
	}

	if err := a.session.AppendEvent(ctx, getResp.Session, event); err != nil {
		slog.Error("saveVoiceTurn: session.AppendEvent() error", "role", role, "err", err)
	}
}

func (a *Assistant) saveVoiceAudio(ctx context.Context, userID int, sessionID, role string, audioChunks [][]byte, sampleRate uint32) int {
	if len(audioChunks) == 0 {
		return 0
	}

	var totalLen int
	for _, chunk := range audioChunks {
		totalLen += len(chunk)
	}
	pcm := make([]byte, 0, totalLen)
	for _, chunk := range audioChunks {
		pcm = append(pcm, chunk...)
	}

	fileName := fmt.Sprintf("voice_%s_%d.wav", role, time.Now().UnixMilli())
	asset, err := tools.SaveChatAudioAsset(ctx, a.db, a.fileStore, userID, sessionID, fileName, buildWAV(pcm, sampleRate, 1, 16), "audio/wav")
	if err != nil {
		slog.Error("saveVoiceAudio: SaveChatAudioAsset failed", "err", err)
		return 0
	}
	slog.Info("saveVoiceAudio: saved", "role", role, "fileID", asset.ID)
	return asset.ID
}

func writeWSError(conn *websocket.Conn, errMsg string) {
	data, _ := json.Marshal(wsServerMessage{Type: serverMsgTypeError, Data: errMsg})
	conn.WriteMessage(websocket.TextMessage, data)
}
