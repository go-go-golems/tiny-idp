package idpprogram

import "sort"

// DiagnosticSeverity is stable output for compilers, CLIs, and audit records.
type DiagnosticSeverity string

const (
	SeverityError   DiagnosticSeverity = "error"
	SeverityWarning DiagnosticSeverity = "warning"
)

// Diagnostic is one deterministic validation finding.
type Diagnostic struct {
	ID       string             `json:"id"`
	Severity DiagnosticSeverity `json:"severity"`
	Path     string             `json:"path"`
	Message  string             `json:"message"`
}

// Diagnostics is a sortable validation result.
type Diagnostics []Diagnostic

// HasErrors reports whether validation produced any error diagnostic.
func (d Diagnostics) HasErrors() bool {
	for _, diagnostic := range d {
		if diagnostic.Severity == SeverityError {
			return true
		}
	}
	return false
}

func (d Diagnostics) sorted() Diagnostics {
	ret := append(Diagnostics(nil), d...)
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].Path != ret[j].Path {
			return ret[i].Path < ret[j].Path
		}
		if ret[i].ID != ret[j].ID {
			return ret[i].ID < ret[j].ID
		}
		return ret[i].Message < ret[j].Message
	})
	return ret
}
