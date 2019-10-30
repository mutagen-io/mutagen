package tunneling

// ErrorServerity indicates the severity of an error.
type ErrorSeverity uint8

const (
	// ErrorSeverityRecoverable indicates that an operation can recover from the
	// associated error and should be retried immediately.
	ErrorSeverityRecoverable ErrorSeverity = iota
	// ErrorSeverityDelayedRecoverable indicates that an operation can recover
	// from the associated error, but should only be retried after some period
	// of time.
	ErrorSeverityDelayedRecoverable
	// ErrorSeverityUnrecoverable indicates that an operation cannot recover
	// from the associated error.
	ErrorSeverityUnrecoverable
)
