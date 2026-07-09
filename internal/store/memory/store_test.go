package memory_test

import (
	"testing"

	"github.com/manuel/tinyidp/internal/store/memory"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

func TestStoreSuite(t *testing.T) {
	idpstore.RunStoreSuite(t, func(t *testing.T) idpstore.Store { return memory.New() })
}
