package semanticmatcher

var _ Logger = (*DiscardLogger)(nil)

// DiscardLogger is a logger that does nothing
type DiscardLogger struct{}

func (DiscardLogger) Debug(...any) {}

func (DiscardLogger) Info(...any) {}

func (DiscardLogger) Warn(...any) {}

func (DiscardLogger) Error(...any) {}

func (DiscardLogger) Debugf(string, ...any) {}

func (DiscardLogger) Infof(string, ...any) {}

func (DiscardLogger) Warnf(string, ...any) {}

func (DiscardLogger) Errorf(string, ...any) {}
