// Package assurance defines the versioned, dependency-free identifiers shared
// by Tiny-IDP configuration, native transitions, verification scenarios, and
// secret-free runtime observations. It deliberately imports no protocol,
// persistence, Goja, or HTTP package.
package assurance

import "strings"

const VocabularyVersion = "tinyidp.assurance/v1"

type (
	HandlerID     string
	SchemaID      string
	CapabilityID  string
	EffectID      string
	EvidenceID    string
	DiagnosticID  string
	ObservationID string
	OutcomeID     string
	StepID        string
	PropertyID    string
)

const (
	OutcomeContinue  OutcomeID = "continue"
	OutcomePresent   OutcomeID = "present"
	OutcomeChallenge OutcomeID = "challenge"
	OutcomeCommit    OutcomeID = "commit"
	OutcomeComplete  OutcomeID = "complete"
	OutcomeDeny      OutcomeID = "deny"
	OutcomeSkip      OutcomeID = "skip"
	OutcomeError     OutcomeID = "error"

	ObservationLambdaStarted   ObservationID = "lambda.started@v1"
	ObservationLambdaCompleted ObservationID = "lambda.completed@v1"
	ObservationLambdaRejected  ObservationID = "lambda.rejected@v1"

	PropertyAuthorizationRequiresNativeCommit PropertyID = "authorization.requires_native_commit@v1"
)

// ValidStableID accepts the bounded ASCII identifier vocabulary used in
// diagnostics, evidence, handler names, and machine-readable observations. It
// deliberately excludes user-controlled free text and whitespace.
func ValidStableID(value string) bool {
	if len(value) == 0 || len(value) > 128 || strings.TrimSpace(value) != value {
		return false
	}
	versionAt := strings.IndexByte(value, '@')
	if versionAt >= 0 {
		if strings.Count(value, "@") != 1 || versionAt == 0 || versionAt+2 >= len(value) || value[versionAt+1] != 'v' {
			return false
		}
		for _, r := range value[versionAt+2:] {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-' || r == '@' {
			continue
		}
		return false
	}
	return true
}
