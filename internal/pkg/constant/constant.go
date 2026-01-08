package constant

type AppName string

func (a AppName) String() string {
	return string(a)
}

const (
	AppNameTravel       AppName = "travel"
	AppNameWebSummary   AppName = "web_summary"
	AppNameEmailSummary AppName = "email_summary"
	AppNameAssistant    AppName = "assistant"
	AppNameImageGen     AppName = "imagegen"
)
