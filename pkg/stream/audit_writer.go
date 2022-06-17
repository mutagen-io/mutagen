package stream

import (
	"io"
)

// Auditor is a callback type that receives written byte counts from write
// operations. Audiot implementations should be fast and minimal to avoid any
// impact on performance.
type Auditor func(uint64)

// auditWriter is an io.Writer that implements write operation auditing.
type auditWriter struct {
	// writer is the underlying writer.
	writer io.Writer
	// auditor is the auditing callback.
	auditor Auditor
}

// NewAuditWriter creates a new io.Writer that invokes an auditing callback with
// written byte counts. If auditor is nil, then this function will return writer
// unmodified.
func NewAuditWriter(writer io.Writer, auditor Auditor) io.Writer {
	if auditor == nil {
		return writer
	}
	return &auditWriter{writer, auditor}
}

// Write implements io.Writer.Write.
func (w *auditWriter) Write(buffer []byte) (int, error) {
	result, err := w.writer.Write(buffer)
	w.auditor(uint64(result))
	return result, err
}
