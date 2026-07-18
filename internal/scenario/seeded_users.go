package scenario

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/go-go-golems/tiny-idp/internal/user"
	"gopkg.in/yaml.v3"
)

// SeededUserFile is the on-disk shape accepted by --users-file.
//
// Example:
//
//	users:
//	  - login: alice
//	    sub: user-alice
//	    email: alice@example.test
//	    name: Alice Inbox
//	    email_verified: true
//	    claims:
//	      groups: [inbox-users]
//	      tenant: personal
//
// The top-level may also be a bare list of users for compact test fixtures.
type SeededUserFile struct {
	Users []SeededUser `json:"users" yaml:"users"`
}

// SeededUser describes a deterministic test principal registered as a normal
// scenario. Missing sub/email/name values are derived from Login, exactly like
// arbitrary fallback users. Claims are merged into both ID token and userinfo.
type SeededUser struct {
	Login    string `json:"login" yaml:"login"`
	Sub      string `json:"sub" yaml:"sub"`
	Email    string `json:"email" yaml:"email"`
	Name     string `json:"name" yaml:"name"`
	Password string `json:"password" yaml:"password"`

	// Support both spellings because YAML fixtures often prefer kebab-case while
	// JSON/OIDC examples usually prefer snake_case.
	EmailVerified      *bool `json:"email_verified" yaml:"email_verified"`
	EmailVerifiedKebab *bool `json:"email-verified" yaml:"email-verified"`

	Groups            []string `json:"groups" yaml:"groups"`
	Roles             []string `json:"roles" yaml:"roles"`
	Tenant            string   `json:"tenant" yaml:"tenant"`
	PreferredUsername string   `json:"preferred_username" yaml:"preferred_username"`
	Locale            string   `json:"locale" yaml:"locale"`

	Claims     map[string]any `json:"claims" yaml:"claims"`
	OmitClaims []string       `json:"omit_claims" yaml:"omit_claims"`
	Category   string         `json:"category" yaml:"category"`
}

// LoadSeededUsers reads a JSON or YAML users file and converts it to scenarios.
// The file extension is intentionally ignored; YAML is a superset for the
// simple structures used here, and JSON is attempted first for clearer JSON
// errors when a JSON fixture is malformed.
func LoadSeededUsers(path string) ([]Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read users file %q: %w", path, err)
	}

	var wrapped SeededUserFile
	jsonErr := json.Unmarshal(data, &wrapped)
	if jsonErr == nil && len(wrapped.Users) > 0 {
		return SeededUsersToScenarios(wrapped.Users)
	}

	wrapped = SeededUserFile{}
	yamlErr := yaml.Unmarshal(data, &wrapped)
	if yamlErr == nil && len(wrapped.Users) > 0 {
		return SeededUsersToScenarios(wrapped.Users)
	}

	var bare []SeededUser
	bareErr := yaml.Unmarshal(data, &bare)
	if bareErr == nil && len(bare) > 0 {
		return SeededUsersToScenarios(bare)
	}

	if jsonErr != nil || yamlErr != nil || bareErr != nil {
		return nil, fmt.Errorf("decode users file %q as JSON or YAML: json: %v; yaml object: %v; yaml list: %v", path, jsonErr, yamlErr, bareErr)
	}
	return nil, fmt.Errorf("decode users file %q: no users found", path)
}

// SeededUsersToScenarios converts configured users into normal scenarios.
func SeededUsersToScenarios(users []SeededUser) ([]Scenario, error) {
	out := make([]Scenario, 0, len(users))
	for i, su := range users {
		sc, err := seededUserToScenario(su)
		if err != nil {
			return nil, fmt.Errorf("users[%d]: %w", i, err)
		}
		out = append(out, sc)
	}
	return out, nil
}

func seededUserToScenario(su SeededUser) (Scenario, error) {
	login := user.Normalize(su.Login)
	if login == "" {
		return Scenario{}, fmt.Errorf("login is required")
	}

	u := user.FromLogin(login)
	if strings.TrimSpace(su.Sub) != "" {
		u.Sub = strings.TrimSpace(su.Sub)
	}
	if strings.TrimSpace(su.Email) != "" {
		u.Email = strings.TrimSpace(su.Email)
	}
	if strings.TrimSpace(su.Name) != "" {
		u.Name = strings.TrimSpace(su.Name)
	}

	extra := genericClaimPresets(su)
	for k, v := range su.Claims {
		extra[k] = v
	}
	if ev := firstBool(su.EmailVerified, su.EmailVerifiedKebab); ev != nil {
		extra["email_verified"] = *ev
	}

	category := strings.TrimSpace(su.Category)
	if category == "" {
		category = "Seeded users"
	}

	return Scenario{
		Name:        login,
		Description: "seeded user from users file",
		Category:    category,
		User:        u,
		Password:    strings.TrimSpace(su.Password),
		ExtraClaims: extra,
		OmitClaims:  append([]string(nil), su.OmitClaims...),
	}, nil
}

func genericClaimPresets(su SeededUser) map[string]any {
	extra := map[string]any{}
	if groups := cleanStringList(su.Groups); len(groups) > 0 {
		extra["groups"] = groups
	}
	if roles := cleanStringList(su.Roles); len(roles) > 0 {
		extra["roles"] = roles
	}
	if tenant := cleanString(su.Tenant); tenant != "" {
		extra["tenant"] = tenant
	}
	if preferredUsername := cleanString(su.PreferredUsername); preferredUsername != "" {
		extra["preferred_username"] = preferredUsername
	}
	if locale := cleanString(su.Locale); locale != "" {
		extra["locale"] = locale
	}
	return extra
}

func cleanString(value string) string {
	return strings.TrimSpace(value)
}

func cleanStringList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := cleanString(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func firstBool(values ...*bool) *bool {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

// RegisterAll adds or replaces scenarios on the registry. It is used by the
// CLI after loading a users file so seeded users can override builtins such as
// alice and bob with stable test subjects.
func (r *Registry) RegisterAll(scenarios []Scenario) {
	for _, sc := range scenarios {
		r.Register(sc)
	}
}
