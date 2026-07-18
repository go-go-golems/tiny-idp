package fositeadapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

func TestPersistBrowserSessionRefreshesSubjectAndBoundsRememberedAccounts(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	userOne := idpstore.User{ID: "u1", Sub: "subject-one", Name: "One"}
	userTwo := idpstore.User{ID: "u2", Sub: "subject-two", Name: "Two"}
	if err := store.PutUser(ctx, "one", userOne); err != nil {
		t.Fatal(err)
	}
	if err := store.PutUser(ctx, "two", userTwo); err != nil {
		t.Fatal(err)
	}
	secret := []byte("account-chooser-persistence-secret-key")
	provider := &Provider{
		store:   store,
		csrfKey: secret,
		chooser: AccountChooserConfig{
			Enabled:                 true,
			ContextCookieName:       defaultBrowserContextCookieName,
			ContextTTL:              24 * time.Hour,
			MaxRememberedAccounts:   1,
			RememberOnPasswordLogin: true,
			DisplayLabel:            func(user idpstore.User) (string, error) { return user.Name, nil },
		},
	}
	now := time.Date(2026, time.July, 14, 18, 0, 0, 0, time.UTC)
	firstRequest := httptest.NewRequest(http.MethodPost, "https://issuer.example.test/authorize", nil)
	first := idpstore.Session{IDHash: []byte("first-session"), UserID: userOne.ID, AuthTime: now, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Hour)}
	contextHandle, remembered, err := provider.persistBrowserSession(firstRequest, userOne, first, now)
	if err != nil || !remembered || contextHandle == "" {
		t.Fatalf("first persistence context=%q remembered=%v err=%v", contextHandle, remembered, err)
	}

	secondRequest := httptest.NewRequest(http.MethodPost, "https://issuer.example.test/authorize", nil)
	secondRequest.AddCookie(&http.Cookie{Name: defaultBrowserContextCookieName, Value: contextHandle})
	second := idpstore.Session{IDHash: []byte("second-session"), UserID: userOne.ID, AuthTime: now.Add(time.Minute), CreatedAt: now.Add(time.Minute), LastSeenAt: now.Add(time.Minute), ExpiresAt: now.Add(time.Hour)}
	newContextHandle, remembered, err := provider.persistBrowserSession(secondRequest, userOne, second, now.Add(time.Minute))
	if err != nil || !remembered || newContextHandle != "" {
		t.Fatalf("same-user refresh context=%q remembered=%v err=%v", newContextHandle, remembered, err)
	}
	contextHash := idpstore.HashSecret(secret, contextHandle)
	entries, err := store.ListRememberedBrowserSessions(ctx, contextHash, now.Add(time.Minute))
	if err != nil || len(entries) != 1 || entries[0].UserID != userOne.ID || string(entries[0].SessionIDHash) != string(second.IDHash) {
		t.Fatalf("same-user refresh entries=%#v err=%v", entries, err)
	}

	third := idpstore.Session{IDHash: []byte("third-session"), UserID: userTwo.ID, AuthTime: now.Add(2 * time.Minute), CreatedAt: now.Add(2 * time.Minute), LastSeenAt: now.Add(2 * time.Minute), ExpiresAt: now.Add(time.Hour)}
	_, remembered, err = provider.persistBrowserSession(secondRequest, userTwo, third, now.Add(2*time.Minute))
	if err != nil || !remembered {
		t.Fatalf("bounded replacement remembered=%v err=%v", remembered, err)
	}
	entries, err = store.ListRememberedBrowserSessions(ctx, contextHash, now.Add(2*time.Minute))
	if err != nil || len(entries) != 1 || entries[0].UserID != userTwo.ID || string(entries[0].SessionIDHash) != string(third.IDHash) {
		t.Fatalf("bounded replacement entries=%#v err=%v", entries, err)
	}
	if source, err := store.GetSession(ctx, second.IDHash); err != nil || source.RevokedAt != nil {
		t.Fatalf("removing remembered membership revoked source session: %#v err=%v", source, err)
	}
}
