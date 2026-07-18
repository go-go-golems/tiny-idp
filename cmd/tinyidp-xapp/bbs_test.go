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

func TestBBSValidationAndRoutingMatrix(t *testing.T) {
	source, err := os.ReadFile("app/objects/objects.js")
	if err != nil {
		t.Fatal(err)
	}
	server := newTestBBSServer(t, t.TempDir(), string(source))
	alice := map[string]any{"actorId": "actor-alice", "actorName": "Alice"}

	postCases := []struct {
		name string
		body map[string]any
		want string
	}{
		{name: "title type", body: map[string]any{"title": 42, "body": "body", "category": "general"}, want: "title_must_be_text"},
		{name: "title empty", body: map[string]any{"title": " \t", "body": "body", "category": "general"}, want: "title_required"},
		{name: "title length", body: map[string]any{"title": strings.Repeat("t", 101), "body": "body", "category": "general"}, want: "title_too_long"},
		{name: "body type", body: map[string]any{"title": "title", "body": true, "category": "general"}, want: "body_must_be_text"},
		{name: "body empty", body: map[string]any{"title": "title", "body": "\n", "category": "general"}, want: "body_required"},
		{name: "body length", body: map[string]any{"title": "title", "body": strings.Repeat("b", 4001), "category": "general"}, want: "body_too_long"},
		{name: "category type", body: map[string]any{"title": "title", "body": "body", "category": []string{"general"}}, want: "category_must_be_text"},
		{name: "category empty", body: map[string]any{"title": "title", "body": "body", "category": " "}, want: "category_required"},
		{name: "category length", body: map[string]any{"title": "title", "body": "body", "category": strings.Repeat("c", 25)}, want: "category_too_long"},
		{name: "category value", body: map[string]any{"title": "title", "body": "body", "category": "private"}, want: "invalid_category"},
	}
	for _, tt := range postCases {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]any{"actorId": alice["actorId"], "actorName": alice["actorName"]}
			for key, value := range tt.body {
				body[key] = value
			}
			response := dispatchTestBBSRaw(t, server, "POST", "/posts", body)
			if response.Status != 400 || responseError(t, response) != tt.want {
				t.Fatalf("response = %#v, want status=400 error=%s", response, tt.want)
			}
		})
	}

	unchanged := dispatchTestBBS(t, server, "GET", "/board", alice, 200)
	if unchanged.Stats.Posts != 0 || unchanged.Stats.Replies != 0 {
		t.Fatalf("invalid post inputs mutated board: %#v", unchanged)
	}

	created := dispatchTestBBS(t, server, "POST", "/posts", map[string]any{
		"actorId": "actor-alice", "actorName": "Alice", "title": "Valid", "body": "body", "category": "QUESTIONS",
	}, 201)
	if created.Posts[0].ID != "post_000000000001" {
		t.Fatalf("invalid requests consumed post sequence: %#v", created)
	}
	if created.Posts[0].Title != "Valid" {
		t.Fatalf("valid post title = %q", created.Posts[0].Title)
	}

	replyCases := []struct {
		name string
		body any
		want string
	}{
		{name: "type", body: 42, want: "body_must_be_text"},
		{name: "empty", body: " ", want: "body_required"},
		{name: "length", body: strings.Repeat("r", 2001), want: "body_too_long"},
	}
	for _, tt := range replyCases {
		t.Run("reply "+tt.name, func(t *testing.T) {
			response := dispatchTestBBSRaw(t, server, "POST", "/posts/post_000000000001/replies", map[string]any{
				"actorId": "actor-alice", "actorName": "Alice", "body": tt.body,
			})
			if response.Status != 400 || responseError(t, response) != tt.want {
				t.Fatalf("response = %#v, want status=400 error=%s", response, tt.want)
			}
		})
	}

	for _, tt := range []struct {
		method string
		path   string
	}{
		{method: "POST", path: "/posts/post_999999999999/replies"},
		{method: "DELETE", path: "/posts/post_999999999999"},
		{method: "POST", path: "/posts/not-an-id/replies"},
		{method: "PATCH", path: "/board"},
	} {
		response := dispatchTestBBSRaw(t, server, tt.method, tt.path, map[string]any{
			"actorId": "actor-alice", "actorName": "Alice", "body": "reply",
		})
		if response.Status != 404 || responseError(t, response) != "post_not_found" && responseError(t, response) != "not_found" {
			t.Fatalf("%s %s response = %#v", tt.method, tt.path, response)
		}
	}

	final := dispatchTestBBS(t, server, "GET", "/board", alice, 200)
	if final.Stats.Posts != 1 || final.Stats.Replies != 0 {
		t.Fatalf("invalid reply and route inputs mutated board: %#v", final)
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
