package constant

type AppName string

func (a AppName) String() string {
	return string(a)
}

const (
	AppNameSearch AppName = "search"
)
