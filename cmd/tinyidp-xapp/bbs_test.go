package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/go-go-objects/pkg/durableobjects"
)

type testBBSBoard struct {
	Name  string `json:"name"`
	Posts []struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Author    string `json:"author"`
		CanDelete bool   `json:"canDelete"`
		Replies   []struct {
			ID     string `json:"id"`
			Body   string `json:"body"`
			Author string `json:"author"`
		} `json:"replies"`
	} `json:"posts"`
	Stats struct {
		Posts   int `json:"posts"`
		Replies int `json:"replies"`
	} `json:"stats"`
}

func TestBBSSharedStateOwnershipAndRestart(t *testing.T) {
	source, err := os.ReadFile("app/objects/objects.js")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	server := newTestBBSServer(t, root, string(source))

	alice := map[string]any{"actorId": "actor-alice", "actorName": "Alice"}
	bob := map[string]any{"actorId": "actor-bob", "actorName": "Bob"}

	initial := dispatchTestBBS(t, server, "GET", "/board", alice, 200)
	if initial.Name != "Local Loop" || initial.Stats.Posts != 0 {
		t.Fatalf("initial board = %#v", initial)
	}

	invalid := dispatchTestBBSRaw(t, server, "POST", "/posts", map[string]any{
		"actorId": "actor-alice", "actorName": "Alice", "title": " ", "body": "body", "category": "general",
	})
	if invalid.Status != 400 || responseError(t, invalid) != "title_required" {
		t.Fatalf("invalid response = %#v", invalid)
	}

	created := dispatchTestBBS(t, server, "POST", "/posts", map[string]any{
		"actorId": "actor-alice", "actorName": "Alice", "title": " First post ", "body": "Shared state", "category": "projects",
	}, 201)
	if len(created.Posts) != 1 || created.Posts[0].ID != "post_000000000001" || created.Posts[0].Title != "First post" || created.Posts[0].Author != "Alice" || !created.Posts[0].CanDelete {
		t.Fatalf("created board = %#v", created)
	}
	publicResponse := dispatchTestBBSRaw(t, server, "GET", "/board", alice)
	createdJSON, err := json.Marshal(publicResponse.Body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(createdJSON), "actor-alice") || strings.Contains(string(createdJSON), "authorId") {
		t.Fatalf("public projection leaked actor identity: %s", createdJSON)
	}

	bobView := dispatchTestBBS(t, server, "GET", "/board", bob, 200)
	if len(bobView.Posts) != 1 || bobView.Posts[0].CanDelete {
		t.Fatalf("bob view = %#v", bobView)
	}

	replied := dispatchTestBBS(t, server, "POST", "/posts/post_000000000001/replies", map[string]any{
		"actorId": "actor-bob", "actorName": "Bob", "body": "Reply from Bob",
	}, 201)
	if len(replied.Posts[0].Replies) != 1 || replied.Posts[0].Replies[0].ID != "reply_000000000001" || replied.Posts[0].Replies[0].Author != "Bob" {
		t.Fatalf("replied board = %#v", replied)
	}

	denied := dispatchTestBBSRaw(t, server, "DELETE", "/posts/post_000000000001", bob)
	if denied.Status != 403 || responseError(t, denied) != "not_post_author" {
		t.Fatalf("bob delete response = %#v", denied)
	}

	if err := server.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	server = newTestBBSServer(t, root, string(source))
	persisted := dispatchTestBBS(t, server, "GET", "/board", alice, 200)
	if persisted.Stats.Posts != 1 || persisted.Stats.Replies != 1 || len(persisted.Posts[0].Replies) != 1 {
		t.Fatalf("persisted board = %#v", persisted)
	}

	deleted := dispatchTestBBS(t, server, "DELETE", "/posts/post_000000000001", alice, 200)
	if deleted.Stats.Posts != 0 || deleted.Stats.Replies != 0 {
		t.Fatalf("deleted board = %#v", deleted)
	}
}

func newTestBBSServer(t *testing.T, root, source string) *durableobjects.Server {
	t.Helper()
	server, err := durableobjects.NewServer(context.Background(), durableobjects.ServerOptions{
		BundleSource: source,
		StorageRoot:  root,
		CPUTimeout:   2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = server.Close(context.Background())
	})
	return server
}

func dispatchTestBBS(t *testing.T, server *durableobjects.Server, method, path string, body map[string]any, wantStatus int) testBBSBoard {
	t.Helper()
	response := dispatchTestBBSRaw(t, server, method, path, body)
	if response.Status != wantStatus {
		t.Fatalf("%s %s status = %d, want %d; body=%#v", method, path, response.Status, wantStatus, response.Body)
	}
	encoded, err := json.Marshal(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	var board testBBSBoard
	if err := json.Unmarshal(encoded, &board); err != nil {
		t.Fatal(err)
	}
	return board
}

func dispatchTestBBSRaw(t *testing.T, server *durableobjects.Server, method, path string, body map[string]any) *durableobjects.FetchResponse {
	t.Helper()
	id, err := durableobjects.NewObjectID("BBS", "community")
	if err != nil {
		t.Fatal(err)
	}
	result, err := server.Manager.Dispatch(context.Background(), durableobjects.Envelope{
		Kind: durableobjects.KindFetch,
		ID:   id,
		Request: &durableobjects.FetchRequest{
			Method: method,
			Path:   path,
			Body:   body,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Response == nil {
		t.Fatal("BBS dispatch returned no response")
	}
	return result.Response
}

func responseError(t *testing.T, response *durableobjects.FetchResponse) string {
	t.Helper()
	encoded, err := json.Marshal(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatal(err)
	}
	return body.Error
}
