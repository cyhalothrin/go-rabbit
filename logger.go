package rabbit

// logger you can set you own logger to session
type logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
}

type noopLogger struct{}

func (n *noopLogger) Debug(_ ...interface{}) {}

func (n *noopLogger) Info(_ ...interface{}) {}

func (n *noopLogger) Warn(_ ...interface{}) {}

type logWrapper struct {
	log       logger
	debugMode bool
}

func newLogWrapper(log logger, debugMode bool) *logWrapper {
	return &logWrapper{
		log:       log,
		debugMode: debugMode,
	}
}

func (l *logWrapper) Debug(args ...interface{}) {
	if l.debugMode {
		l.log.Debug(args...)
	}
}

func (l *logWrapper) Info(args ...interface{}) {
	l.log.Info(args...)
}

func (l *logWrapper) Warn(args ...interface{}) {
	l.log.Warn(args...)
}
