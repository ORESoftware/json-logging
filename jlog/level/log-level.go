package ll

type LogLevel int

const (
	TRACE LogLevel = iota
	DEBUG LogLevel = iota
	INFO  LogLevel = iota
	WARN  LogLevel = iota
	ERROR LogLevel = iota
)
