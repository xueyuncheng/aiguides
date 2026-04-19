package assistant

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"google.golang.org/genai"
)

const (
	// TTSModel is the Gemini TTS model identifier.
	TTSModel = "gemini-2.5-flash-preview-tts"
	// TTSMaxTextLength is the max character limit for TTS requests.
	TTSMaxTextLength = 10000
	// TTSMaxChunkLen is the max characters per sentence chunk sent to Gemini.
	TTSMaxChunkLen = 200
)

// TTSRequest defines the request body for text-to-speech conversion.
type TTSRequest struct {
	Text      string `json:"text"`
	VoiceName string `json:"voice_name,omitempty"` // e.g. "Kore", "Puck", "Charon". Defaults to "Kore".
}

// TextToSpeechStream splits the input text into sentence-level chunks, calls
// Gemini TTS for each chunk sequentially, and streams the resulting WAV audio
// back to the client as Server-Sent Events.
//
// Each SSE event has type "chunk" and a JSON payload:
//
//	{"index": <int>, "data": "<base64-encoded WAV>"}
//
// A final event of type "done" is sent when all chunks have been processed.
// On error, an event of type "error" is sent with {"error": "<message>"}.
//
// POST /api/assistant/tts/stream
func (a *Assistant) TextToSpeechStream(ctx *gin.Context) {
	// 1. Authenticate
	userID, ok := getContextUserID(ctx)
	if !ok || userID <= 0 {
		slog.Error("TextToSpeechStream: invalid or missing user_id in context")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 2. Parse and validate request
	var req TTSRequest
	if err := ctx.BindJSON(&req); err != nil {
		slog.Error("TextToSpeechStream: ctx.BindJSON() error", "err", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "text cannot be empty"})
		return
	}
	if len(req.Text) > TTSMaxTextLength {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("text too long (max %d characters)", TTSMaxTextLength)})
		return
	}
	if req.VoiceName == "" {
		req.VoiceName = "Kore"
	}

	// 3. Split text into sentence chunks
	chunks := splitSentences(req.Text, TTSMaxChunkLen)
	if len(chunks) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "text cannot be empty"})
		return
	}

	slog.Debug("TextToSpeechStream: starting",
		"chunks", len(chunks),
		"voice", req.VoiceName,
		"user_id", userID,
	)

	// 4. Set up SSE response
	ctx.Writer.Header().Set("Content-Type", "text/event-stream")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Connection", "keep-alive")
	ctx.Writer.Header().Set("Content-Encoding", "none")
	ctx.Writer.Flush()

	// 5. Generate and stream each chunk
	for i, chunk := range chunks {
		// Check if client disconnected
		select {
		case <-ctx.Request.Context().Done():
			slog.Debug("TextToSpeechStream: client disconnected", "chunk", i)
			return
		default:
		}

		wavData, err := a.generateTTSChunk(ctx, chunk, req.VoiceName)
		if err != nil {
			slog.Error("TextToSpeechStream: generateTTSChunk error", "chunk", i, "err", err)
			ctx.SSEvent("error", gin.H{"error": fmt.Sprintf("failed to generate chunk %d: %s", i, err.Error())})
			ctx.Writer.Flush()
			return
		}

		slog.Info("TextToSpeechStream: chunk ready",
			"index", i,
			"wav_bytes", len(wavData),
			"user_id", userID,
		)

		ctx.SSEvent("chunk", gin.H{
			"index": i,
			"total": len(chunks),
			"data":  base64.StdEncoding.EncodeToString(wavData),
		})
		ctx.Writer.Flush()
	}

	ctx.SSEvent("done", gin.H{"total": len(chunks)})
	ctx.Writer.Flush()

	slog.Info("TextToSpeechStream: completed",
		"user_id", userID,
		"chunks", len(chunks),
	)
}

// generateTTSChunk calls Gemini TTS for a single text chunk and returns WAV bytes.
func (a *Assistant) generateTTSChunk(ctx *gin.Context, text, voiceName string) ([]byte, error) {
	ttsResp, err := a.genaiClient.Models.GenerateContent(
		ctx,
		TTSModel,
		genai.Text(text),
		&genai.GenerateContentConfig{
			ResponseModalities: []string{"AUDIO"},
			SpeechConfig: &genai.SpeechConfig{
				VoiceConfig: &genai.VoiceConfig{
					PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{
						VoiceName: voiceName,
					},
				},
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("genaiClient.Models.GenerateContent() error: %w", err)
	}

	if len(ttsResp.Candidates) == 0 || ttsResp.Candidates[0].Content == nil {
		return nil, fmt.Errorf("empty candidates from Gemini TTS")
	}

	var rawData []byte
	responseMIME := ""
	for _, part := range ttsResp.Candidates[0].Content.Parts {
		if part.InlineData != nil && len(part.InlineData.Data) > 0 {
			rawData = part.InlineData.Data
			responseMIME = part.InlineData.MIMEType
			break
		}
	}

	if len(rawData) == 0 {
		return nil, fmt.Errorf("no audio data in Gemini TTS response")
	}

	return pcmToWAV(rawData, responseMIME), nil
}

// splitSentences splits text into chunks at sentence boundaries (。！？!?\n),
// keeping each chunk under maxLen characters. If a sentence exceeds maxLen it
// is split at the nearest word/character boundary.
func splitSentences(text string, maxLen int) []string {
	// Sentence-ending punctuation (Chinese and ASCII).
	isSentenceEnd := func(r rune) bool {
		switch r {
		case '。', '！', '？', '!', '?', '\n', '…':
			return true
		}
		return false
	}

	var chunks []string
	runes := []rune(text)
	start := 0

	for start < len(runes) {
		// Find the next sentence boundary within maxLen.
		end := start
		lastBoundary := -1

		for end < len(runes) && end-start < maxLen {
			if isSentenceEnd(runes[end]) {
				lastBoundary = end + 1 // include the punctuation
			}
			end++
		}

		var chunk string
		if end >= len(runes) {
			// Reached end of text.
			chunk = string(runes[start:end])
			start = end
		} else if lastBoundary > start {
			// Cut at the last sentence boundary within maxLen.
			chunk = string(runes[start:lastBoundary])
			start = lastBoundary
		} else {
			// No sentence boundary found; cut at a space or hard-cut.
			cutAt := start + maxLen
			for cutAt > start && !unicode.IsSpace(runes[cutAt-1]) {
				cutAt--
			}
			if cutAt == start {
				cutAt = start + maxLen // no space found, hard-cut
			}
			chunk = string(runes[start:cutAt])
			start = cutAt
		}

		chunk = strings.TrimSpace(chunk)
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
	}

	return chunks
}

// pcmToWAV parses sample-rate / channel info from the Gemini MIME type and
// wraps the raw PCM bytes in a standard RIFF/WAV container.
// If the data is already a known container format it is returned unchanged.
func pcmToWAV(raw []byte, mimeType string) []byte {
	lower := strings.ToLower(mimeType)

	// Already a container format — pass through as-is.
	if strings.Contains(lower, "wav") ||
		strings.Contains(lower, "mp3") || strings.Contains(lower, "mpeg") ||
		strings.Contains(lower, "ogg") ||
		strings.Contains(lower, "aac") {
		return raw
	}

	// Parse "audio/l16; rate=24000; channels=1"
	sampleRate := uint32(24000)
	numChannels := uint16(1)

	for _, param := range strings.Split(mimeType, ";") {
		param = strings.TrimSpace(param)
		if v, ok := strings.CutPrefix(param, "rate="); ok {
			if n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 32); err == nil {
				sampleRate = uint32(n)
			}
		}
		if v, ok := strings.CutPrefix(param, "channels="); ok {
			if n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 16); err == nil {
				numChannels = uint16(n)
			}
		}
	}

	return buildWAV(raw, sampleRate, numChannels, 16)
}

// buildWAV wraps raw signed-16-bit little-endian PCM in a RIFF/WAV container.
func buildWAV(pcm []byte, sampleRate uint32, numChannels uint16, bitsPerSample uint16) []byte {
	const headerSize = 44
	byteRate := sampleRate * uint32(numChannels) * uint32(bitsPerSample) / 8
	blockAlign := numChannels * bitsPerSample / 8
	dataSize := uint32(len(pcm))
	chunkSize := uint32(headerSize-8) + dataSize

	buf := make([]byte, headerSize+len(pcm))

	// RIFF chunk descriptor
	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:8], chunkSize)
	copy(buf[8:12], "WAVE")

	// fmt sub-chunk
	copy(buf[12:16], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:20], 16) // PCM fmt chunk is always 16 bytes
	binary.LittleEndian.PutUint16(buf[20:22], 1)  // audio format: 1 = PCM
	binary.LittleEndian.PutUint16(buf[22:24], numChannels)
	binary.LittleEndian.PutUint32(buf[24:28], sampleRate)
	binary.LittleEndian.PutUint32(buf[28:32], byteRate)
	binary.LittleEndian.PutUint16(buf[32:34], blockAlign)
	binary.LittleEndian.PutUint16(buf[34:36], bitsPerSample)

	// data sub-chunk
	copy(buf[36:40], "data")
	binary.LittleEndian.PutUint32(buf[40:44], dataSize)
	copy(buf[44:], pcm)

	return buf
}
