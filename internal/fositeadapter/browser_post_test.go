package fositeadapter

import (
	"crypto/tls"
	"net/http"
	"testing"
)

func TestSameOriginBrowserPost(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		tls     bool
		headers map[string]string
		want    bool
	}{
		{
			name:   "explicit HTTP origin",
			origin: "http://idp.localhost:8443",
			want:   true,
		},
		{
			name:   "explicit HTTPS origin",
			origin: "https://idp.localhost:8443",
			tls:    true,
			want:   true,
		},
		{
			name:   "explicit origin contradicted by cross-site metadata",
			origin: "https://idp.localhost:8443",
			tls:    true,
			headers: map[string]string{
				"Sec-Fetch-Site": "cross-site",
			},
		},
		{
			name:   "cross-site explicit origin",
			origin: "https://attacker.example.test",
			tls:    true,
			headers: map[string]string{
				"Sec-Fetch-Site": "cross-site",
			},
		},
		{
			name:   "Firefox null origin form navigation",
			origin: "null",
			tls:    true,
			headers: map[string]string{
				"Sec-Fetch-Site": "same-origin",
				"Sec-Fetch-Mode": "navigate",
				"Sec-Fetch-Dest": "document",
				"Sec-Fetch-User": "?1",
			},
			want: true,
		},
		{
			name:   "null origin without Fetch Metadata",
			origin: "null",
			tls:    true,
		},
		{
			name:   "null origin from cross-site navigation",
			origin: "null",
			tls:    true,
			headers: map[string]string{
				"Sec-Fetch-Site": "cross-site",
				"Sec-Fetch-Mode": "navigate",
				"Sec-Fetch-Dest": "document",
				"Sec-Fetch-User": "?1",
			},
		},
		{
			name:   "null origin from a same-origin fetch",
			origin: "null",
			tls:    true,
			headers: map[string]string{
				"Sec-Fetch-Site": "same-origin",
				"Sec-Fetch-Mode": "cors",
				"Sec-Fetch-Dest": "empty",
				"Sec-Fetch-User": "?1",
			},
		},
		{
			name:   "null origin without user activation",
			origin: "null",
			tls:    true,
			headers: map[string]string{
				"Sec-Fetch-Site": "same-origin",
				"Sec-Fetch-Mode": "navigate",
				"Sec-Fetch-Dest": "document",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := http.NewRequest(http.MethodPost, "http://idp.localhost:8443/authorize", nil)
			if err != nil {
				t.Fatal(err)
			}
			r.Host = "idp.localhost:8443"
			if tt.tls {
				r.TLS = &tls.ConnectionState{}
			}
			r.Header.Set("Origin", tt.origin)
			for name, value := range tt.headers {
				r.Header.Set(name, value)
			}
			if got := sameOriginBrowserPost(r); got != tt.want {
				t.Fatalf("sameOriginBrowserPost() = %v, want %v", got, tt.want)
			}
		})
	}
}
