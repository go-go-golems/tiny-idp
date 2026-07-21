package idpui

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxBrowserErrorTitleRunes   = 120
	maxBrowserErrorHeadingRunes = 160
	maxBrowserErrorSummaryRunes = 500
)

// BrowserErrorPage is a terminal, presentation-only error document. It
// intentionally has no form, action, continuation, credential, or redirect
// fields. ClientID is public context used to select an approved server-side
// theme.
type BrowserErrorPage struct {
	DocumentTitle string
	ClientID      string
	Heading       string
	Summary       string
}

func (p BrowserErrorPage) Validate() error {
	for _, field := range []struct {
		name  string
		value string
		limit int
	}{
		{name: "document title", value: p.DocumentTitle, limit: maxBrowserErrorTitleRunes},
		{name: "client ID", value: p.ClientID, limit: 256},
		{name: "heading", value: p.Heading, limit: maxBrowserErrorHeadingRunes},
		{name: "summary", value: p.Summary, limit: maxBrowserErrorSummaryRunes},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is required", field.name)
		}
		if strings.TrimSpace(field.value) != field.value || utf8.RuneCountInString(field.value) > field.limit {
			return fmt.Errorf("%s is not canonical or exceeds %d characters", field.name, field.limit)
		}
	}
	return nil
}
