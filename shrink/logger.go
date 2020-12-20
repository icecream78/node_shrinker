package shrink

type Logger interface {
	Infof(format string, a ...interface{})
	Infoln(a ...interface{})
}
