package assistant

const (
	clientMsgTypeAudio = "audio"
	clientMsgTypeEnd   = "end"
	clientMsgTypeText  = "text"
)

const (
	serverMsgTypeSetupOK          = "setup_ok"
	serverMsgTypeAudio            = "audio"
	serverMsgTypeInputTranscript  = "input_transcript"
	serverMsgTypeOutputTranscript = "output_transcript"
	serverMsgTypeTurnComplete     = "turn_complete"
	serverMsgTypeInterrupted      = "interrupted"
	serverMsgTypeError            = "error"
	serverMsgTypeGoAway           = "go_away"
	serverMsgTypeToolCallStart    = "tool_call_start"
	serverMsgTypeToolCallEnd      = "tool_call_end"
)

type wsClientMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
}

type wsServerMessage struct {
	Type     string `json:"type"`
	Data     string `json:"data,omitempty"`
	Finished bool   `json:"finished,omitempty"`
}
