package tools

import (
	"aiguide/internal/app/aiguide/table"
	"aiguide/internal/pkg/constant"
	"aiguide/internal/pkg/middleware"
	"aiguide/internal/pkg/storage"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
	"gorm.io/gorm"
)

const (
	defaultAudioModel                = "gemini-3.1-flash-lite-preview"
	maxAudioTranscriptChars          = 20000
	directAudioTranscriptionLimitMs  = int64(10 * 60 * 1000)
	maxSyncAudioTranscriptionLimitMs = int64(45 * 60 * 1000)
	defaultAudioChunkDurationMs      = int64(5 * 60 * 1000)
	defaultAudioChunkOverlapMs       = int64(5 * 1000)
	audioFileReadyPollInterval       = 2 * time.Second
	audioFileReadyPollTimeout        = 2 * time.Minute
)

var supportedAudioMimeTypes = map[string]bool{
	"audio/mpeg":     true,
	"audio/mp3":      true,
	"audio/wav":      true,
	"audio/x-wav":    true,
	"audio/wave":     true,
	"audio/x-pn-wav": true,
	"audio/mp4":      true,
	"audio/x-m4a":    true,
	"audio/aac":      true,
	"audio/webm":     true,
	"audio/ogg":      true,
}

const ContextKeyAudioTranscriptionProgressReporter = "audio_transcription_progress_reporter"

type AudioTranscriptionProgress struct {
	FileID          int    `json:"file_id"`
	JobID           int    `json:"job_id"`
	ChunkIndex      int    `json:"chunk_index"`
	CompletedChunks int    `json:"completed_chunks"`
	ChunkCount      int    `json:"chunk_count"`
	StartMs         int64  `json:"start_ms"`
	EndMs           int64  `json:"end_ms"`
	Transcript      string `json:"transcript"`
	Characters      int    `json:"characters"`
}

type AudioTranscriptionProgressReporter func(AudioTranscriptionProgress)

type AudioTranscribeInput struct {
	FileID int    `json:"file_id" jsonschema:"ID of the uploaded audio file to transcribe"`
	Prompt string `json:"prompt,omitempty" jsonschema:"Optional hint for names, jargon, or expected language"`
}

type AudioTranscribeOutput struct {
	Success    bool   `json:"success"`
	FileID     int    `json:"file_id"`
	JobID      int    `json:"job_id"`
	Language   string `json:"language,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	Transcript string `json:"transcript"`
	Truncated  bool   `json:"truncated"`
	Characters int    `json:"characters"`
	ChunkCount int    `json:"chunk_count"`
	Message    string `json:"message"`
}

type AudioToolService struct {
	db          *gorm.DB
	fileStore   storage.FileStore
	genaiClient *genai.Client
	workDir     string
}

type audioChunkWindow struct {
	index          int
	startMs        int64
	endMs          int64
	overlapStartMs int64
	overlapEndMs   int64
}

func IsSupportedAudioMimeType(mimeType string) bool {
	_, ok := supportedAudioMimeTypes[strings.ToLower(strings.TrimSpace(mimeType))]
	return ok
}

func NewAudioTranscribeTool(db *gorm.DB, fileStore storage.FileStore, genaiClient *genai.Client, workDir string) (tool.Tool, error) {
	service, err := newAudioToolService(db, fileStore, genaiClient, workDir)
	if err != nil {
		return nil, err
	}

	config := functiontool.Config{
		Name:        "audio_transcribe",
		Description: "Transcribe a user-owned audio file to plain text. Long audio is chunked automatically.",
	}

	handler := func(ctx tool.Context, input AudioTranscribeInput) (*AudioTranscribeOutput, error) {
		return service.transcribe(ctx, input)
	}

	return functiontool.New(config, handler)
}

func SaveChatAudioAsset(ctx context.Context, db *gorm.DB, fileStore storage.FileStore, userID int, sessionID, fileName string, data []byte, mimeType string) (*table.FileAsset, error) {
	if db == nil {
		slog.Error("db is nil")
		return nil, fmt.Errorf("database connection is required")
	}
	if fileStore == nil {
		slog.Error("fileStore is nil")
		return nil, fmt.Errorf("file store is required")
	}
	if !IsSupportedAudioMimeType(mimeType) {
		slog.Error("invalid audio mime type", "mime_type", mimeType)
		return nil, fmt.Errorf("unsupported audio type: %s", mimeType)
	}

	meta, err := fileStore.Save(ctx, storage.SaveInput{
		UserID:    userID,
		SessionID: sessionID,
		FileName:  fileName,
		MimeType:  mimeType,
		Content:   bytes.NewReader(data),
		SizeBytes: int64(len(data)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store uploaded audio: %w", err)
	}

	asset := &table.FileAsset{
		UserID:       userID,
		SessionID:    sessionID,
		Kind:         constant.FileAssetKindUploaded,
		MimeType:     mimeType,
		OriginalName: fileName,
		StoragePath:  meta.StoragePath,
		SizeBytes:    meta.SizeBytes,
		SHA256:       meta.SHA256,
		Status:       constant.FileAssetStatusReady,
		TextStatus:   constant.PDFTextExtractStatusPending,
	}

	if err := db.Create(asset).Error; err != nil {
		slog.Error("db.Create() error", "err", err, "file_name", fileName)
		return nil, fmt.Errorf("failed to persist uploaded audio asset: %w", err)
	}

	return asset, nil
}

func newAudioToolService(db *gorm.DB, fileStore storage.FileStore, genaiClient *genai.Client, workDir string) (*AudioToolService, error) {
	if db == nil {
		slog.Error("db is nil")
		return nil, fmt.Errorf("database connection is required")
	}
	if fileStore == nil {
		slog.Error("fileStore is nil")
		return nil, fmt.Errorf("file store is required")
	}
	if strings.TrimSpace(workDir) == "" {
		slog.Error("workDir is empty")
		return nil, fmt.Errorf("audio work directory is required")
	}
	root := filepath.Join(filepath.Clean(workDir), "audio")
	if err := os.MkdirAll(root, 0755); err != nil {
		slog.Error("os.MkdirAll() error", "work_dir", root, "err", err)
		return nil, fmt.Errorf("os.MkdirAll() error: %w", err)
	}

	return &AudioToolService{
		db:          db,
		fileStore:   fileStore,
		genaiClient: genaiClient,
		workDir:     root,
	}, nil
}

func (s *AudioToolService) transcribe(ctx context.Context, input AudioTranscribeInput) (*AudioTranscribeOutput, error) {
	if s.genaiClient == nil {
		slog.Error("genai client is nil")
		return nil, fmt.Errorf("genai client is required for audio transcription")
	}

	userID, sessionID, err := s.requireContext(ctx)
	if err != nil {
		return nil, err
	}

	asset, err := s.loadAudioAsset(userID, input.FileID)
	if err != nil {
		return nil, err
	}

	job := &table.AudioJob{
		UserID:    userID,
		SessionID: sessionID,
		FileID:    asset.ID,
		Status:    constant.AudioJobStatusPending,
		ModelName: defaultAudioModel,
		MimeType:  asset.MimeType,
		Prompt:    strings.TrimSpace(input.Prompt),
	}
	if err := s.db.Create(job).Error; err != nil {
		slog.Error("db.Create() error", "file_id", input.FileID, "err", err)
		return nil, fmt.Errorf("failed to create audio job: %w", err)
	}

	if err := s.markJobRunning(job.ID); err != nil {
		return nil, err
	}

	workspace, cleanup, err := s.createWorkspace(userID, sessionID, job.ID)
	if err != nil {
		s.failJob(job.ID, err)
		return nil, err
	}
	defer cleanup()

	inputPath := filepath.Join(workspace, "input"+audioFileExtension(asset.OriginalName, asset.MimeType))
	if err := s.materializeAsset(ctx, asset, inputPath); err != nil {
		s.failJob(job.ID, err)
		return nil, err
	}

	durationMs, err := probeAudioDuration(inputPath)
	if err != nil {
		s.failJob(job.ID, err)
		return nil, err
	}
	if durationMs > maxSyncAudioTranscriptionLimitMs {
		err = fmt.Errorf("audio is too long for synchronous transcription; max supported length is %d minutes", maxSyncAudioTranscriptionLimitMs/(60*1000))
		s.failJob(job.ID, err)
		return nil, err
	}

	normalizedPath := filepath.Join(workspace, "normalized.wav")
	if err := normalizeAudio(inputPath, normalizedPath); err != nil {
		s.failJob(job.ID, err)
		return nil, err
	}

	windows := buildAudioChunkWindows(durationMs)
	if err := s.db.Model(&table.AudioJob{}).Where("id = ?", job.ID).Updates(map[string]any{
		"duration_ms": durationMs,
		"chunk_count": len(windows),
	}).Error; err != nil {
		slog.Error("db.Model().Updates() error", "job_id", job.ID, "err", err)
		s.failJob(job.ID, err)
		return nil, fmt.Errorf("failed to update audio job metadata: %w", err)
	}

	transcripts := make([]string, 0, len(windows))
	mergedTranscript := ""
	for _, window := range windows {
		chunkPath := normalizedPath
		if len(windows) > 1 {
			chunkPath = filepath.Join(workspace, fmt.Sprintf("chunk-%02d.wav", window.index))
			if err := extractAudioChunk(normalizedPath, chunkPath, window.startMs, window.endMs-window.startMs); err != nil {
				s.persistChunkFailure(job.ID, window, err)
				s.failJob(job.ID, err)
				return nil, err
			}
		}

		chunkText, err := s.transcribeChunk(ctx, chunkPath, window, input.Prompt)
		if err != nil {
			s.persistChunkFailure(job.ID, window, err)
			s.failJob(job.ID, err)
			return nil, err
		}
		if err := s.persistChunkSuccess(job.ID, window, chunkText); err != nil {
			s.failJob(job.ID, err)
			return nil, err
		}
		transcripts = append(transcripts, chunkText)
		mergedTranscript = joinChunkTranscripts(transcripts)
		reportAudioTranscriptionProgress(ctx, AudioTranscriptionProgress{
			FileID:          asset.ID,
			JobID:           job.ID,
			ChunkIndex:      window.index,
			CompletedChunks: len(transcripts),
			ChunkCount:      len(windows),
			StartMs:         window.startMs,
			EndMs:           window.endMs,
			Transcript:      mergedTranscript,
			Characters:      len(mergedTranscript),
		})
	}

	fullTranscript := mergedTranscript
	truncatedTranscript := fullTranscript
	truncated := false
	if len(truncatedTranscript) > maxAudioTranscriptChars {
		truncatedTranscript = truncatedTranscript[:maxAudioTranscriptChars]
		truncated = true
	}

	summary := fmt.Sprintf("Transcribed %d audio chunk(s) from %s", len(windows), asset.OriginalName)
	if err := s.db.Model(&table.AudioJob{}).Where("id = ?", job.ID).Updates(map[string]any{
		"status":           constant.AudioJobStatusCompleted,
		"duration_ms":      durationMs,
		"chunk_count":      len(windows),
		"transcript_text":  fullTranscript,
		"transcript_chars": len(fullTranscript),
		"completed_at":     time.Now(),
		"error_message":    "",
	}).Error; err != nil {
		slog.Error("db.Model().Updates() error", "job_id", job.ID, "err", err)
		return nil, fmt.Errorf("failed to complete audio job: %w", err)
	}

	return &AudioTranscribeOutput{
		Success:    true,
		FileID:     asset.ID,
		JobID:      job.ID,
		DurationMs: durationMs,
		Transcript: truncatedTranscript,
		Truncated:  truncated,
		Characters: len(fullTranscript),
		ChunkCount: len(windows),
		Message:    summary,
	}, nil
}

func (s *AudioToolService) transcribeChunk(ctx context.Context, path string, window audioChunkWindow, userPrompt string) (string, error) {
	uploadedFile, err := s.genaiClient.Files.UploadFromPath(ctx, path, &genai.UploadFileConfig{
		MIMEType:    "audio/wav",
		DisplayName: filepath.Base(path),
	})
	if err != nil {
		slog.Error("Files.UploadFromPath() error", "path", path, "err", err)
		return "", fmt.Errorf("failed to upload audio chunk: %w", err)
	}
	defer func() {
		if uploadedFile == nil || uploadedFile.Name == "" {
			return
		}
		if _, err := s.genaiClient.Files.Delete(context.Background(), uploadedFile.Name, nil); err != nil {
			slog.Warn("Files.Delete() error", "file_name", uploadedFile.Name, "err", err)
		}
	}()

	readyFile, err := s.waitForUploadedFile(ctx, uploadedFile.Name)
	if err != nil {
		return "", err
	}

	contents := []*genai.Content{genai.NewContentFromParts([]*genai.Part{
		genai.NewPartFromText(buildAudioTranscriptionPrompt(userPrompt, window)),
		genai.NewPartFromFile(*readyFile),
	}, genai.RoleUser)}

	resp, err := s.genaiClient.Models.GenerateContent(ctx, defaultAudioModel, contents, nil)
	if err != nil {
		slog.Error("Models.GenerateContent() error", "path", path, "err", err)
		return "", fmt.Errorf("failed to transcribe audio chunk: %w", err)
	}

	text := strings.TrimSpace(resp.Text())
	if text == "" {
		return "", fmt.Errorf("audio chunk transcription returned empty text")
	}
	return text, nil
}

func (s *AudioToolService) waitForUploadedFile(ctx context.Context, fileName string) (*genai.File, error) {
	deadline := time.Now().Add(audioFileReadyPollTimeout)
	for {
		file, err := s.genaiClient.Files.Get(ctx, fileName, nil)
		if err != nil {
			slog.Error("Files.Get() error", "file_name", fileName, "err", err)
			return nil, fmt.Errorf("failed to inspect uploaded audio file: %w", err)
		}
		switch file.State {
		case genai.FileStateActive, genai.FileStateUnspecified:
			return file, nil
		case genai.FileStateFailed:
			if file.Error != nil && file.Error.Message != "" {
				return nil, fmt.Errorf("uploaded audio file failed processing: %s", file.Error.Message)
			}
			return nil, fmt.Errorf("uploaded audio file failed processing")
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for uploaded audio file to become ready")
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("audio file processing cancelled: %w", ctx.Err())
		case <-time.After(audioFileReadyPollInterval):
		}
	}
}

func (s *AudioToolService) persistChunkSuccess(jobID int, window audioChunkWindow, transcript string) error {
	row := &table.AudioTranscriptChunk{
		JobID:           jobID,
		ChunkIndex:      window.index,
		StartMs:         window.startMs,
		EndMs:           window.endMs,
		OverlapStartMs:  window.overlapStartMs,
		OverlapEndMs:    window.overlapEndMs,
		Status:          constant.AudioTranscriptChunkStatusCompleted,
		TranscriptText:  transcript,
		TranscriptChars: len(transcript),
	}
	if err := s.db.Create(row).Error; err != nil {
		slog.Error("db.Create() error", "job_id", jobID, "chunk_index", window.index, "err", err)
		return fmt.Errorf("failed to persist audio chunk transcript: %w", err)
	}
	return nil
}

func (s *AudioToolService) persistChunkFailure(jobID int, window audioChunkWindow, cause error) {
	if cause == nil {
		return
	}
	row := &table.AudioTranscriptChunk{
		JobID:          jobID,
		ChunkIndex:     window.index,
		StartMs:        window.startMs,
		EndMs:          window.endMs,
		OverlapStartMs: window.overlapStartMs,
		OverlapEndMs:   window.overlapEndMs,
		Status:         constant.AudioTranscriptChunkStatusFailed,
		ErrorMessage:   cause.Error(),
	}
	if err := s.db.Create(row).Error; err != nil {
		slog.Warn("db.Create() error", "job_id", jobID, "chunk_index", window.index, "err", err)
	}
}

func (s *AudioToolService) requireContext(ctx context.Context) (int, string, error) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok || userID <= 0 {
		slog.Error("user_id not found in context")
		return 0, "", fmt.Errorf("user_id not found in context")
	}

	sessionID, ok := ctx.Value(constant.ContextKeySessionID).(string)
	if !ok || strings.TrimSpace(sessionID) == "" {
		slog.Error("session_id not found in context")
		return 0, "", fmt.Errorf("session_id not found in context")
	}

	return userID, sessionID, nil
}

func (s *AudioToolService) loadAudioAsset(userID, fileID int) (*table.FileAsset, error) {
	var asset table.FileAsset
	if err := s.db.Where("id = ? AND user_id = ?", fileID, userID).First(&asset).Error; err != nil {
		slog.Error("db.First() error", "file_id", fileID, "user_id", userID, "err", err)
		return nil, fmt.Errorf("failed to load file asset: %w", err)
	}
	if !IsSupportedAudioMimeType(asset.MimeType) {
		slog.Error("file asset is not audio", "file_id", fileID, "mime_type", asset.MimeType)
		return nil, fmt.Errorf("file %d is not an audio file", fileID)
	}
	if asset.Status != constant.FileAssetStatusReady {
		slog.Error("file asset is not ready", "file_id", fileID, "status", asset.Status)
		return nil, fmt.Errorf("file %d is not ready", fileID)
	}
	return &asset, nil
}

func (s *AudioToolService) markJobRunning(jobID int) error {
	now := time.Now()
	if err := s.db.Model(&table.AudioJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":     constant.AudioJobStatusRunning,
		"started_at": now,
	}).Error; err != nil {
		slog.Error("db.Model().Updates() error", "job_id", jobID, "err", err)
		return fmt.Errorf("failed to mark audio job running: %w", err)
	}
	return nil
}

func (s *AudioToolService) failJob(jobID int, cause error) {
	if cause == nil {
		return
	}
	now := time.Now()
	if err := s.db.Model(&table.AudioJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":        constant.AudioJobStatusFailed,
		"error_message": cause.Error(),
		"completed_at":  now,
	}).Error; err != nil {
		slog.Error("db.Model().Updates() error", "job_id", jobID, "err", err)
	}
}

func (s *AudioToolService) createWorkspace(userID int, sessionID string, jobID int) (string, func(), error) {
	workspace := filepath.Join(s.workDir, fmt.Sprintf("%d", userID), sessionID, fmt.Sprintf("%d-%s", jobID, uuid.NewString()))
	if err := os.MkdirAll(workspace, 0755); err != nil {
		slog.Error("os.MkdirAll() error", "workspace", workspace, "err", err)
		return "", nil, fmt.Errorf("os.MkdirAll() error: %w", err)
	}

	cleanup := func() {
		if err := os.RemoveAll(workspace); err != nil {
			slog.Warn("os.RemoveAll() error", "workspace", workspace, "err", err)
		}
	}

	return workspace, cleanup, nil
}

func (s *AudioToolService) materializeAsset(ctx context.Context, asset *table.FileAsset, destination string) error {
	rc, err := s.fileStore.Open(ctx, asset.StoragePath)
	if err != nil {
		return fmt.Errorf("failed to open stored file: %w", err)
	}
	defer rc.Close()

	file, err := os.Create(destination)
	if err != nil {
		slog.Error("os.Create() error", "path", destination, "err", err)
		return fmt.Errorf("os.Create() error: %w", err)
	}

	if _, err := io.Copy(file, rc); err != nil {
		file.Close()
		slog.Error("io.Copy() error", "path", destination, "err", err)
		return fmt.Errorf("io.Copy() error: %w", err)
	}
	if err := file.Close(); err != nil {
		slog.Error("file.Close() error", "path", destination, "err", err)
		return fmt.Errorf("file.Close() error: %w", err)
	}

	return nil
}

func buildAudioChunkWindows(durationMs int64) []audioChunkWindow {
	if durationMs <= 0 {
		return nil
	}
	if durationMs <= directAudioTranscriptionLimitMs {
		return []audioChunkWindow{{
			index:   0,
			startMs: 0,
			endMs:   durationMs,
		}}
	}

	stepMs := defaultAudioChunkDurationMs - defaultAudioChunkOverlapMs
	windows := make([]audioChunkWindow, 0, int(math.Ceil(float64(durationMs)/float64(stepMs))))
	for startMs, idx := int64(0), 0; startMs < durationMs; startMs, idx = startMs+stepMs, idx+1 {
		endMs := startMs + defaultAudioChunkDurationMs
		if endMs > durationMs {
			endMs = durationMs
		}
		window := audioChunkWindow{
			index:   idx,
			startMs: startMs,
			endMs:   endMs,
		}
		if idx > 0 {
			window.overlapStartMs = startMs
		}
		if endMs < durationMs {
			window.overlapEndMs = endMs
		}
		windows = append(windows, window)
	}
	return windows
}

func joinChunkTranscripts(transcripts []string) string {
	if len(transcripts) == 0 {
		return ""
	}
	joined := strings.TrimSpace(transcripts[0])
	for _, transcript := range transcripts[1:] {
		joined = mergeTranscriptText(joined, transcript)
	}
	return joined
}

func mergeTranscriptText(existing, next string) string {
	existing = strings.TrimSpace(existing)
	next = strings.TrimSpace(next)
	if existing == "" {
		return next
	}
	if next == "" {
		return existing
	}

	const minOverlapChars = 40
	const maxOverlapProbe = 400
	existingProbe := existing
	nextProbe := next
	if len(existingProbe) > maxOverlapProbe {
		existingProbe = existingProbe[len(existingProbe)-maxOverlapProbe:]
	}
	if len(nextProbe) > maxOverlapProbe {
		nextProbe = nextProbe[:maxOverlapProbe]
	}

	max := len(existingProbe)
	if len(nextProbe) < max {
		max = len(nextProbe)
	}
	for overlap := max; overlap >= minOverlapChars; overlap-- {
		if existingProbe[len(existingProbe)-overlap:] == nextProbe[:overlap] {
			return existing + next[overlap:]
		}
	}

	return existing + "\n\n" + next
}

func buildAudioTranscriptionPrompt(userPrompt string, window audioChunkWindow) string {
	base := []string{
		"Generate an accurate transcript of the audio.",
		"",
		"Requirements:",
		"1. Output plain text only.",
		"2. Preserve the original language.",
		"3. Do not summarize.",
		"4. Do not rewrite for style.",
		"5. If any portion is unclear, mark it as [inaudible].",
	}
	if userPrompt = strings.TrimSpace(userPrompt); userPrompt != "" {
		base = append(base, "", "Additional context:", userPrompt)
	}
	if window.startMs > 0 || window.endMs > 0 {
		base = append(base, "", fmt.Sprintf("This chunk covers approximately %s to %s of the full audio.", formatAudioTimestamp(window.startMs), formatAudioTimestamp(window.endMs)))
	}
	return strings.Join(base, "\n")
}

func formatAudioTimestamp(ms int64) string {
	totalSeconds := ms / 1000
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func audioFileExtension(fileName, mimeType string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext != "" {
		return ext
	}
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/wav", "audio/x-wav", "audio/wave", "audio/x-pn-wav":
		return ".wav"
	case "audio/mp4", "audio/x-m4a":
		return ".m4a"
	case "audio/webm":
		return ".webm"
	case "audio/ogg":
		return ".ogg"
	default:
		return ".audio"
	}
}

func probeAudioDuration(path string) (int64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	output, err := cmd.Output()
	if err != nil {
		slog.Error("ffprobe error", "path", path, "err", err)
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}
	durationSeconds, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		slog.Error("strconv.ParseFloat() error", "output", strings.TrimSpace(string(output)), "err", err)
		return 0, fmt.Errorf("invalid ffprobe duration output: %w", err)
	}
	return int64(durationSeconds * 1000), nil
}

func normalizeAudio(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, "-ac", "1", "-ar", "16000", outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("ffmpeg normalize error", "input_path", inputPath, "output_path", outputPath, "output", strings.TrimSpace(string(output)), "err", err)
		return fmt.Errorf("ffmpeg normalize failed: %w", err)
	}
	return nil
}

func extractAudioChunk(inputPath, outputPath string, startMs, durationMs int64) error {
	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-ss", formatFFmpegSeconds(startMs),
		"-t", formatFFmpegSeconds(durationMs),
		"-i", inputPath,
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("ffmpeg chunk error", "input_path", inputPath, "output_path", outputPath, "output", strings.TrimSpace(string(output)), "err", err)
		return fmt.Errorf("ffmpeg chunk extraction failed: %w", err)
	}
	return nil
}

func formatFFmpegSeconds(ms int64) string {
	seconds := float64(ms) / 1000
	return strconv.FormatFloat(seconds, 'f', 3, 64)
}

func reportAudioTranscriptionProgress(ctx context.Context, progress AudioTranscriptionProgress) {
	if ctx == nil {
		return
	}
	reporter, ok := ctx.Value(ContextKeyAudioTranscriptionProgressReporter).(AudioTranscriptionProgressReporter)
	if !ok || reporter == nil {
		return
	}
	reporter(progress)
}
