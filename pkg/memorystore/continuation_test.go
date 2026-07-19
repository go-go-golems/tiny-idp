package memorystore_test

import (
	"testing"

	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation"
	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation/idpcontinuationtest"
	"github.com/go-go-golems/tiny-idp/pkg/memorystore"
)

func TestContinuationStoreSuite(t *testing.T) {
	idpcontinuationtest.RunStoreSuite(t, func(*testing.T) idpcontinuation.Store {
		return memorystore.NewContinuationStore()
	})
}
