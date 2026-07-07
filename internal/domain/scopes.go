package domain

import "strings"

func ParseScopes(scope string) []string {
	fields := strings.Fields(scope)
	seen := map[string]struct{}{}
	out := make([]string, 0, len(fields))
	for _, s := range fields {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func HasScope(scopes []string, want string) bool {
	for _, s := range scopes {
		if s == want {
			return true
		}
	}
	return false
}
