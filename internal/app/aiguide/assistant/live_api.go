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
	"strings"
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

	historyContext := a.buildHistoryContext(connCtx, userIDStr, sessionID)

	liveSession, err := a.connectLiveSession(connCtx, userID, historyContext)
	if err != nil {
		writeWSError(conn, err.Error())
		return
	}
	defer liveSession.Close()

	writeCh := make(chan wsServerMessage, 32)
	defer close(writeCh)
	go wsWriter(conn, writeCh, connCancel)

	sessionWriteCh := make(chan func() error, 32)
	defer close(sessionWriteCh)
	go sessionWriter(liveSession, sessionWriteCh, connCancel)

	writeCh <- wsServerMessage{Type: serverMsgTypeSetupOK}

	saveTurn, saveImageTurn, waitSaves := a.startSaveLoop(userID, sessionID)
	defer waitSaves()

	firstErr := a.bridgeVoiceCall(connCtx, connCancel, writeCh, liveSession, conn, saveTurn, saveImageTurn, userID, sessionID, sessionWriteCh)

	if firstErr != nil {
		slog.Info("VoiceCall: session ended with error", "reason", firstErr.Error(), "userID", userID)
		// Send error to client before the deferred close(writeCh) shuts down wsWriter.
		writeCh <- wsServerMessage{Type: serverMsgTypeError, Data: firstErr.Error()}
	} else {
		slog.Info("VoiceCall: session ended normally", "userID", userID)
	}
}

func (a *Assistant) connectLiveSession(ctx context.Context, userID int, historyContext string) (*genai.Session, error) {
	liveClient, err := a.getLiveClient(ctx)
	if err != nil {
		slog.Error("VoiceCall: getLiveClient failed", "err", err, "userID", userID)
		return nil, fmt.Errorf("failed to initialize live client: %w", err)
	}

	liveModel := a.liveModel
	if liveModel == "" {
		liveModel = defaultLiveModel
	}

	systemInstruction := assistantAgentInstruction + "\n\n## 当前模式\n你正在通过语音通话与用户交互。"
	if historyContext != "" {
		systemInstruction += "\n\n" + historyContext
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
		RealtimeInputConfig: &genai.RealtimeInputConfig{
			AutomaticActivityDetection: &genai.AutomaticActivityDetection{
				StartOfSpeechSensitivity: genai.StartSensitivityLow,
				EndOfSpeechSensitivity:   genai.EndSensitivityHigh,
			},
		},
		InputAudioTranscription:  &genai.AudioTranscriptionConfig{},
		OutputAudioTranscription: &genai.AudioTranscriptionConfig{},
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{genai.NewPartFromText(systemInstruction)},
		},
		Tools: buildGenaiTools(a.liveTools),
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

func sessionWriter(_ *genai.Session, writeCh <-chan func() error, cancel context.CancelFunc) {
	for fn := range writeCh {
		if err := fn(); err != nil {
			slog.Error("VoiceCall: liveSession write error", "err", err)
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
	imageParts []*genai.Part
}

func (a *Assistant) startSaveLoop(userID int, sessionID string) (saveTurn func(string, string, [][]byte, uint32), saveImageTurn func(string, []*genai.Part), wait func()) {
	saveCh := make(chan saveRequest, 16)
	saveDone := make(chan struct{})
	var firstUserTranscript string
	var userTurnCount int
	var titleGenerated bool
	go func() {
		defer close(saveDone)
		for req := range saveCh {
			if len(req.imageParts) > 0 {
				a.saveTextImageTurn(context.Background(), userID, sessionID, req.text, req.imageParts)
			} else {
				a.saveVoiceTurn(context.Background(), userID, sessionID, req.role, req.text, req.audio, req.sampleRate)
			}
			if req.role != "user" || req.text == "" {
				continue
			}
			if firstUserTranscript == "" {
				firstUserTranscript = req.text
			}
			userTurnCount++
			if userTurnCount == 2 && !titleGenerated {
				titleGenerated = true
				transcript := firstUserTranscript
				go func() {
					if err := a.generateTitle(context.Background(), sessionID, transcript); err != nil {
						slog.Error("startSaveLoop: generateTitle (2nd turn) failed", "err", err)
					}
				}()
			}
		}
	}()

	saveTurn = func(role, text string, audioChunks [][]byte, sampleRate uint32) {
		slog.Info("saveTurn: queued", "role", role, "textLen", len(text), "audioChunks", len(audioChunks))
		copied := make([][]byte, len(audioChunks))
		copy(copied, audioChunks)
		saveCh <- saveRequest{role: role, text: text, audio: copied, sampleRate: sampleRate}
	}

	saveImageTurn = func(text string, imageParts []*genai.Part) {
		slog.Info("saveImageTurn: queued", "textLen", len(text), "imageParts", len(imageParts))
		saveCh <- saveRequest{role: "user", text: text, imageParts: imageParts}
	}

	wait = func() {
		close(saveCh)
		<-saveDone
		if firstUserTranscript != "" && !titleGenerated {
			go func() {
				if err := a.generateTitle(context.Background(), sessionID, firstUserTranscript); err != nil {
					slog.Error("startSaveLoop: generateTitle (fallback) failed", "err", err)
				}
			}()
		}
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
	saveImageTurn func(string, []*genai.Part),
	userID int,
	sessionID string,
	sessionWriteCh chan<- func() error,
) error {
	tracker := &turnTracker{phase: phaseWaiting}
	registry := buildLiveToolRegistry(a.liveTools)

	toolCtx := context.WithValue(ctx, constant.ContextKeyUserID, userID)
	toolCtx = context.WithValue(toolCtx, constant.ContextKeySessionID, sessionID)
	toolCtx = context.WithValue(toolCtx, constant.ContextKeyTx, a.db)

	errCh := make(chan error, 2)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("VoiceCall: panic in receiveFromGemini", "panic", r)
				errCh <- fmt.Errorf("panic in receiveFromGemini: %v", r)
			}
		}()
		errCh <- a.receiveFromGeminiWithTracking(ctx, writeCh, liveSession, tracker, saveTurn, registry, toolCtx, sessionWriteCh)
	}()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("VoiceCall: panic in receiveFromClient", "panic", r)
				errCh <- fmt.Errorf("panic in receiveFromClient: %v", r)
			}
		}()
		errCh <- receiveFromClientWithTracking(ctx, conn, liveSession, cancel, tracker, saveTurn, saveImageTurn, sessionWriteCh)
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

func (a *Assistant) receiveFromGeminiWithTracking(ctx context.Context, writeCh chan<- wsServerMessage, liveSession *genai.Session, tracker *turnTracker, saveTurn func(string, string, [][]byte, uint32), registry liveToolRegistry, toolCtx context.Context, sessionWriteCh chan<- func() error) error {
	for {
		msg, err := liveSession.Receive()
		if err != nil {
			tracker.flushPending(saveTurn)
			return fmt.Errorf("session.Receive: %w", err)
		}

		if msg.GoAway != nil {
			writeCh <- wsServerMessage{Type: serverMsgTypeGoAway}
		}

		if msg.ToolCall != nil {
			writeCh <- wsServerMessage{Type: serverMsgTypeToolCallStart}
			responses := make([]*genai.FunctionResponse, 0, len(msg.ToolCall.FunctionCalls))
			for _, call := range msg.ToolCall.FunctionCalls {
				responses = append(responses, executeLiveTool(toolCtx, registry, call, a.genaiClient, a.modelName))
			}
			writeCh <- wsServerMessage{Type: serverMsgTypeToolCallEnd}
			select {
			case sessionWriteCh <- func() error {
				return liveSession.SendToolResponse(genai.LiveSendToolResponseParameters{
					FunctionResponses: responses,
				})
			}:
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
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

func (t *turnTracker) finalizePendingTurns(saveTurn func(string, string, [][]byte, uint32)) {
	t.flushPending(saveTurn)
	switch t.phase {
	case phaseUser:
		if t.userText != "" || len(t.userAudio) > 0 {
			saveTurn("user", t.userText, t.userAudio, 16000)
		}
	case phaseModel:
		if t.modelText != "" || len(t.modelAudio) > 0 {
			saveTurn("model", t.modelText, t.modelAudio, 24000)
		}
	}
	t.phase = phaseWaiting
	t.userAudio = t.userAudio[:0]
	t.userText = ""
	t.modelAudio = t.modelAudio[:0]
	t.modelText = ""
}

func (t *turnTracker) handleTextInput(text string, saveTurn func(string, string, [][]byte, uint32)) {
	t.finalizePendingTurns(saveTurn)
	saveTurn("user", text, nil, 0)
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

func receiveFromClientWithTracking(ctx context.Context, conn *websocket.Conn, liveSession *genai.Session, cancel context.CancelFunc, tracker *turnTracker, saveTurn func(string, string, [][]byte, uint32), saveImageTurn func(string, []*genai.Part), sessionWriteCh chan<- func() error) error {
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
			select {
			case sessionWriteCh <- func() error {
				return liveSession.SendRealtimeInput(genai.LiveRealtimeInput{
					Audio: &genai.Blob{Data: pcm, MIMEType: "audio/pcm;rate=16000"},
				})
			}:
			case <-ctx.Done():
				return nil
			}

		case clientMsgTypeEnd:
			select {
			case sessionWriteCh <- func() error {
				return liveSession.SendRealtimeInput(genai.LiveRealtimeInput{AudioStreamEnd: true})
			}:
			case <-ctx.Done():
				return nil
			}

		case clientMsgTypeText:
			if msg.Data == "" && len(msg.Images) == 0 {
				continue
			}
			text := msg.Data

			var imageParts []*genai.Part
			for _, dataURI := range msg.Images {
				imgBytes, mimeType, err := parseDataURI(dataURI)
				if err != nil {
					slog.Warn("receiveFromClient: image parseDataURI error", "err", err)
					continue
				}
				if !strings.HasPrefix(mimeType, "image/") {
					slog.Warn("receiveFromClient: skipping non-image file", "mimeType", mimeType)
					continue
				}
				imageParts = append(imageParts, genai.NewPartFromBytes(imgBytes, mimeType))
			}

			if len(imageParts) > 0 {
				tracker.finalizePendingTurns(saveTurn)
				saveImageTurn(text, imageParts)
			} else if text != "" {
				tracker.handleTextInput(text, saveTurn)
			} else {
				continue
			}

			parts := make([]*genai.Part, 0, 1+len(imageParts))
			if text != "" {
				parts = append(parts, genai.NewPartFromText(text))
			}
			parts = append(parts, imageParts...)
			select {
			case sessionWriteCh <- func() error {
				return liveSession.SendClientContent(genai.LiveClientContentInput{
					Turns: []*genai.Content{{Role: genai.RoleUser, Parts: parts}},
				})
			}:
			case <-ctx.Done():
				return nil
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

	var messageText string
	if len(audioChunks) > 0 {
		metadataJSON, _ := json.Marshal(voiceAudioMetadata{FileID: fileID})
		messageText = fmt.Sprintf("%s %s -->\n%s", voiceAudioMetadataPrefix, string(metadataJSON), transcript)
	} else {
		messageText = transcript
	}

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
		slog.Error("failed to get session", "err", err, "op", "saveVoiceTurn")
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
		slog.Error("failed to append event", "role", role, "err", err, "op", "saveVoiceTurn")
	}
}

func (a *Assistant) saveTextImageTurn(ctx context.Context, userID int, sessionID, text string, imageParts []*genai.Part) {
	if text == "" && len(imageParts) == 0 {
		return
	}

	slog.Info("saveTextImageTurn: starting", "textLen", len(text), "imageParts", len(imageParts), "sessionID", sessionID)

	parts := make([]*genai.Part, 0, 1+len(imageParts))
	if text != "" {
		parts = append(parts, genai.NewPartFromText(text))
	}
	parts = append(parts, imageParts...)

	userIDStr := strconv.Itoa(userID)
	getResp, err := a.session.Get(ctx, &adksession.GetRequest{
		AppName:   constant.AppNameAssistant.String(),
		UserID:    userIDStr,
		SessionID: sessionID,
	})
	if err != nil {
		slog.Error("failed to get session", "err", err, "op", "saveTextImageTurn")
		return
	}

	event := &adksession.Event{
		ID:        uuid.NewString(),
		Timestamp: time.Now(),
		Author:    "user",
		LLMResponse: adkmodel.LLMResponse{
			Content: &genai.Content{
				Role:  genai.RoleUser,
				Parts: parts,
			},
		},
	}

	if err := a.session.AppendEvent(ctx, getResp.Session, event); err != nil {
		slog.Error("failed to append event", "err", err, "op", "saveTextImageTurn")
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

// maxHistoryTurns is the maximum number of text turns included in the Live session context.
const maxHistoryTurns = 20

// buildHistoryContext loads the text history from the ADK session and formats it
// as a "## 历史对话" block to be appended to the Live session's system instruction.
// Returns an empty string if there is no history or the session cannot be loaded.
// Binary content (images, audio blobs) is skipped — only the textual substance
// is needed for conversational continuity.
func (a *Assistant) buildHistoryContext(ctx context.Context, userIDStr, sessionID string) string {
	getResp, err := a.session.Get(ctx, &adksession.GetRequest{
		AppName:   constant.AppNameAssistant.String(),
		UserID:    userIDStr,
		SessionID: sessionID,
	})
	if err != nil {
		slog.Warn("buildHistoryContext: session.Get failed", "err", err, "sessionID", sessionID)
		return ""
	}

	type turn struct {
		role string
		text string
	}
	var turns []turn

	for event := range getResp.Session.Events().All() {
		if event.Content == nil {
			continue
		}

		var textParts []string
		for _, part := range event.Content.Parts {
			// Skip thoughts, tool calls, tool responses, and binary blobs.
			if part.Thought || part.FunctionCall != nil || part.FunctionResponse != nil || part.InlineData != nil {
				continue
			}
			if part.Text == "" {
				continue
			}
			text := part.Text
			// Voice turns store "<!-- VOICE_AUDIO: {...} -->\nTranscript" — extract transcript only.
			if _, transcript, ok := extractVoiceAudioMetadata(text); ok {
				text = transcript
			}
			text = stripUserContext(strings.TrimSpace(text))
			if text != "" {
				textParts = append(textParts, text)
			}
		}

		if len(textParts) == 0 {
			continue
		}

		role := "助手"
		if event.Content.Role == "user" {
			role = "用户"
		}
		turns = append(turns, turn{role: role, text: strings.Join(textParts, "\n")})
	}

	if len(turns) == 0 {
		return ""
	}

	// Keep only the most recent turns to stay within the system instruction size limit.
	if len(turns) > maxHistoryTurns {
		turns = turns[len(turns)-maxHistoryTurns:]
	}

	slog.Info("buildHistoryContext: built history", "turns", len(turns), "sessionID", sessionID)

	var sb strings.Builder
	sb.WriteString("## 历史对话\n以下是本次对话在切换到语音模式前的文字记录，请以此为上下文继续对话：\n\n")
	for _, t := range turns {
		sb.WriteString(t.role)
		sb.WriteString("：")
		sb.WriteString(t.text)
		sb.WriteString("\n\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}
