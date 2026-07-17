package fositeadapter_test

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anishathalye/porcupine"
	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

var oneTimeConsumeModel = porcupine.Model{
	Init: func() any { return false },
	Step: func(state, input, output any) (bool, any) {
		consumed := state.(bool)
		succeeded := output.(bool)
		if succeeded {
			return !consumed, true
		}
		return consumed, consumed
	},
	DescribeOperation: func(input, output any) string {
		if output.(bool) {
			return "consume -> success"
		}
		return "consume -> rejected"
	},
}

func TestInteractionConsumeHistoryIsLinearizable(t *testing.T) {
	store := memory.New()
	ctx := context.Background()
	now := time.Now().UTC()
	hash := []byte("porcupine-interaction-hash")
	if err := store.CreateInteraction(ctx, idpstore.InteractionRecord{IDHash: hash, CreatedAt: now, ExpiresAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	const workers = 16
	operations := make([]porcupine.Operation, workers)
	start := make(chan struct{})
	var clock atomic.Int64
	var wg sync.WaitGroup
	for worker := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			call := clock.Add(1)
			_, err := store.ConsumeInteraction(ctx, hash, now, idpstore.InteractionOutcomeApproved)
			ret := clock.Add(1)
			operations[worker] = porcupine.Operation{ClientId: worker, Input: struct{}{}, Call: call, Output: err == nil, Return: ret}
		}()
	}
	close(start)
	wg.Wait()
	if !porcupine.CheckOperations(oneTimeConsumeModel, operations) {
		t.Fatal("interaction consume history is not linearizable")
	}
}

func TestSQLiteRefreshRotationHistoryIsLinearizableAndReuseRevokesFamily(t *testing.T) {
	store, server, verifier := newSQLiteTokenFixture(t, nil)
	code := authorizeForCode(t, server.URL, verifier)
	tokens := exchangeCode(t, server.URL, code, verifier)
	oldRefresh := tokens["refresh_token"].(string)

	const workers = 8
	operations := make([]porcupine.Operation, workers)
	start := make(chan struct{})
	var clock atomic.Int64
	var wg sync.WaitGroup
	for worker := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			call := clock.Add(1)
			response, err := http.PostForm(server.URL+"/token", url.Values{
				"grant_type":    {"refresh_token"},
				"client_id":     {"spa"},
				"refresh_token": {oldRefresh},
			})
			succeeded := false
			if err == nil {
				succeeded = response.StatusCode == http.StatusOK
				_ = response.Body.Close()
			}
			ret := clock.Add(1)
			operations[worker] = porcupine.Operation{ClientId: worker, Input: struct{}{}, Call: call, Output: succeeded, Return: ret}
		}()
	}
	close(start)
	wg.Wait()
	if !porcupine.CheckOperations(oneTimeConsumeModel, operations) {
		t.Fatal("refresh rotation history is not linearizable")
	}
	// One rotation wins, but subsequent uses of the old token invoke Fosite's
	// reuse response and revoke the complete family, including the winner.
	assertSQLCount(t, store, `SELECT COUNT(*) FROM fosite_refresh_tokens WHERE active=1`, 0)
	assertSQLCount(t, store, `SELECT COUNT(*) FROM fosite_refresh_tokens`, 1)
	assertSQLCount(t, store, `SELECT COUNT(*) FROM fosite_access_tokens`, 0)
}
