package securityerrorcheck

type store struct{}

func (*store) ConsumeInteraction() error { return nil }
func (*store) RecordConsent() error      { return nil }

func ignored(s *store) {
	_ = s.ConsumeInteraction() // want "result from security-critical operation ConsumeInteraction is ignored"
	s.RecordConsent()          // want "result from security-critical operation RecordConsent is ignored"
}
