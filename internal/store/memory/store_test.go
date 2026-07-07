package memory_test

import (
	"testing"

	"github.com/manuel/tinyidp/internal/storage"
	"github.com/manuel/tinyidp/internal/store/memory"
)

func TestStoreSuite(t *testing.T) {
	storage.RunStoreSuite(t, func(t *testing.T) storage.Store { return memory.New() })
}
