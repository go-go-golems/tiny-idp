package idpaccounts_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/tiny-idp/pkg/idpaccounts"
	"github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
)

func TestPublicConstructorDoesNotRequireInternalTypes(t *testing.T) {
	store, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	service, err := idpaccounts.NewService(store, idpaccounts.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if service == nil {
		t.Fatal("service is nil")
	}
}
