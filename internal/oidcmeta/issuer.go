package oidcmeta

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/manuel/tinyidp/internal/domain"
)

type Issuer struct{ URL *url.URL }

func ParseIssuer(raw string) (Issuer, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return Issuer{}, fmt.Errorf("invalid issuer URL")
	}
	u.Fragment = ""
	u.RawQuery = ""
	return Issuer{URL: u}, nil
}

func ValidateIssuer(raw string, mode domain.Mode) error {
	iss, err := ParseIssuer(raw)
	if err != nil {
		return err
	}
	if mode == domain.ProductionMode && iss.URL.Scheme != "https" {
		return fmt.Errorf("production issuer must use https")
	}
	if iss.URL.Scheme == "http" && mode == domain.DevMode && !isLoopbackHost(iss.URL.Hostname()) {
		return fmt.Errorf("dev http issuer must be loopback")
	}
	return nil
}

func (i Issuer) String() string { return i.URL.String() }

func (i Issuer) Endpoint(path string) string {
	base := strings.TrimRight(i.URL.String(), "/")
	return base + path
}

func (i Issuer) DiscoveryPath() string {
	prefix := strings.TrimRight(i.URL.EscapedPath(), "/")
	if prefix == "/" {
		prefix = ""
	}
	return prefix + "/.well-known/openid-configuration"
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
