package scenario

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSeededUsersToScenariosOverridesIdentityAndClaims(t *testing.T) {
	verified := false
	scenarios, err := SeededUsersToScenarios([]SeededUser{
		{
			Login:         " Alice ",
			Sub:           "user-alice-fixed",
			Email:         "alice@inbox.test",
			Name:          "Alice Inbox",
			EmailVerified: &verified,
			Claims: map[string]any{
				"groups": []string{"inbox-users"},
				"tenant": "personal",
			},
		},
	})
	if err != nil {
		t.Fatalf("SeededUsersToScenarios: %v", err)
	}
	if len(scenarios) != 1 {
		t.Fatalf("len = %d", len(scenarios))
	}
	sc := scenarios[0]
	if sc.Name != "alice" {
		t.Fatalf("Name = %q", sc.Name)
	}
	if sc.User.Sub != "user-alice-fixed" || sc.User.Email != "alice@inbox.test" || sc.User.Name != "Alice Inbox" {
		t.Fatalf("unexpected user: %+v", sc.User)
	}
	if sc.ExtraClaims["email_verified"] != false {
		t.Fatalf("email_verified = %#v", sc.ExtraClaims["email_verified"])
	}
	if sc.ExtraClaims["tenant"] != "personal" {
		t.Fatalf("tenant claim = %#v", sc.ExtraClaims["tenant"])
	}
}

func TestSeededUsersToScenariosExpandsGenericClaimPresets(t *testing.T) {
	scenarios, err := SeededUsersToScenarios([]SeededUser{
		{
			Login:             "Alice",
			Groups:            []string{" inbox-users ", "", "engineering"},
			Roles:             []string{" writer ", "reviewer"},
			Tenant:            " personal ",
			PreferredUsername: " alice ",
			Locale:            " en-US ",
		},
	})
	if err != nil {
		t.Fatalf("SeededUsersToScenarios: %v", err)
	}
	claims := scenarios[0].ExtraClaims
	if !reflect.DeepEqual(claims["groups"], []string{"inbox-users", "engineering"}) {
		t.Fatalf("groups = %#v", claims["groups"])
	}
	if !reflect.DeepEqual(claims["roles"], []string{"writer", "reviewer"}) {
		t.Fatalf("roles = %#v", claims["roles"])
	}
	if claims["tenant"] != "personal" {
		t.Fatalf("tenant = %#v", claims["tenant"])
	}
	if claims["preferred_username"] != "alice" {
		t.Fatalf("preferred_username = %#v", claims["preferred_username"])
	}
	if claims["locale"] != "en-US" {
		t.Fatalf("locale = %#v", claims["locale"])
	}
}

func TestSeededUsersToScenariosExplicitClaimsOverrideGenericPresets(t *testing.T) {
	scenarios, err := SeededUsersToScenarios([]SeededUser{
		{
			Login:  "alice",
			Groups: []string{"preset-group"},
			Roles:  []string{"preset-role"},
			Claims: map[string]any{
				"groups": []string{"explicit-group"},
				"roles":  []string{"explicit-role"},
			},
		},
	})
	if err != nil {
		t.Fatalf("SeededUsersToScenarios: %v", err)
	}
	claims := scenarios[0].ExtraClaims
	if !reflect.DeepEqual(claims["groups"], []string{"explicit-group"}) {
		t.Fatalf("groups = %#v", claims["groups"])
	}
	if !reflect.DeepEqual(claims["roles"], []string{"explicit-role"}) {
		t.Fatalf("roles = %#v", claims["roles"])
	}
}

func TestLoadSeededUsersFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.yaml")
	if err := os.WriteFile(path, []byte(`users:
  - login: bob
    sub: user-bob-fixed
    email: bob@inbox.test
    name: Bob Inbox
    email-verified: true
    groups: [inbox-users]
    roles: [reader]
    tenant: personal
    preferred_username: bob
    locale: en-US
`), 0o644); err != nil {
		t.Fatal(err)
	}
	scenarios, err := LoadSeededUsers(path)
	if err != nil {
		t.Fatalf("LoadSeededUsers: %v", err)
	}
	if len(scenarios) != 1 {
		t.Fatalf("len = %d", len(scenarios))
	}
	sc := scenarios[0]
	if sc.User.Sub != "user-bob-fixed" || sc.User.Email != "bob@inbox.test" {
		t.Fatalf("unexpected user: %+v", sc.User)
	}
	if sc.ExtraClaims["email_verified"] != true {
		t.Fatalf("email_verified = %#v", sc.ExtraClaims["email_verified"])
	}
	if !reflect.DeepEqual(sc.ExtraClaims["groups"], []string{"inbox-users"}) {
		t.Fatalf("groups = %#v", sc.ExtraClaims["groups"])
	}
	if !reflect.DeepEqual(sc.ExtraClaims["roles"], []string{"reader"}) {
		t.Fatalf("roles = %#v", sc.ExtraClaims["roles"])
	}
	if sc.ExtraClaims["tenant"] != "personal" {
		t.Fatalf("tenant = %#v", sc.ExtraClaims["tenant"])
	}
	if sc.ExtraClaims["preferred_username"] != "bob" {
		t.Fatalf("preferred_username = %#v", sc.ExtraClaims["preferred_username"])
	}
	if sc.ExtraClaims["locale"] != "en-US" {
		t.Fatalf("locale = %#v", sc.ExtraClaims["locale"])
	}
}

func TestRegisterAllOverridesBuiltin(t *testing.T) {
	r := New()
	scenarios, err := SeededUsersToScenarios([]SeededUser{{Login: "alice", Sub: "user-alice-fixed"}})
	if err != nil {
		t.Fatal(err)
	}
	r.RegisterAll(scenarios)
	sc, ok := r.Lookup("alice")
	if !ok {
		t.Fatal("expected explicit alice scenario")
	}
	if sc.User.Sub != "user-alice-fixed" {
		t.Fatalf("sub = %q", sc.User.Sub)
	}
}
