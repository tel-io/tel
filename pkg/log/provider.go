package log

type LogProvider interface { //nolint:revive,golint
	Logger() Logger
}
