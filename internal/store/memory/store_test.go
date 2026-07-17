package memory_test

import (
	"testing"

	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

func TestStoreSuite(t *testing.T) {
	idpstore.RunStoreSuite(t, func(t *testing.T) idpstore.Store { return memory.New() })
}
