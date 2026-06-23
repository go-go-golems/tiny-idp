package scenario

import (
	"os"
	"path/filepath"
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

func TestLoadSeededUsersFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users.yaml")
	if err := os.WriteFile(path, []byte(`users:
  - login: bob
    sub: user-bob-fixed
    email: bob@inbox.test
    name: Bob Inbox
    email-verified: true
    claims:
      roles: [reader]
      tenant: personal
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
