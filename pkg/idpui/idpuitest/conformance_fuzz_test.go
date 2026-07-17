package idpuitest_test

import (
	"testing"

	"github.com/go-go-golems/tiny-idp/pkg/idpui/idpuitest"
)

func FuzzConformanceParserNeverPanics(f *testing.F) {
	f.Add([]byte(`<!doctype html><html><body><form></form></body></html>`))
	f.Add([]byte(`<script src="javascript:x"><input type=password value=secret>`))
	f.Add([]byte{0xff, 0xfe, '<', '/', 'f', 'o', 'r', 'm', '>'})
	f.Add(make([]byte, 8192))
	page := conformancePage()
	f.Fuzz(func(t *testing.T, document []byte) {
		if len(document) > 1<<20 {
			t.Skip()
		}
		_, _ = idpuitest.Check(document, page)
	})
}
