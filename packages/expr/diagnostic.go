package expr

// DiagnosticLevel indicates severity.
type DiagnosticLevel int

const (
	DiagError DiagnosticLevel = iota
	DiagWarning
)

// Diagnostic is a compiler message with source location.
type Diagnostic struct {
	Level   DiagnosticLevel
	Span    Span
	Message string
	Code    string
}
