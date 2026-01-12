package constant

const (
	ContextKeyTx string = "tx"
)

type AppName string

func (a AppName) String() string {
	return string(a)
}

const (
	AppNameAssistant AppName = "assistant"
)
