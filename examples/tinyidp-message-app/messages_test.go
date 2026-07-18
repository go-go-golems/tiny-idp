package main

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestMessageCursorPaginationIsStable(t *testing.T) {
	ctx := context.Background()
	store, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	stamp := time.Now().UTC()
	for _, body := range []string{"first", "second", "third"} {
		if _, err := store.createMessage(ctx, message{AuthorSubject: "subject", AuthorName: "Alice", Body: body, CreatedAt: stamp}); err != nil {
			t.Fatal(err)
		}
	}
	firstPage, err := store.listMessages(ctx, nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstPage) != 2 || firstPage[0].Body != "third" || firstPage[1].Body != "second" {
		t.Fatalf("first page = %#v", firstPage)
	}
	secondPage, err := store.listMessages(ctx, &messageCursor{CreatedAt: firstPage[1].CreatedAt, ID: firstPage[1].ID}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondPage) != 1 || secondPage[0].Body != "first" {
		t.Fatalf("second page = %#v", secondPage)
	}
}

func TestMessageValidationAndNormalization(t *testing.T) {
	store, err := openAppStore(context.Background(), filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, value := range []message{
		{AuthorSubject: "", AuthorName: "Name", Body: "body", CreatedAt: time.Now()},
		{AuthorSubject: "subject", AuthorName: "Name", Body: " \n ", CreatedAt: time.Now()},
		{AuthorSubject: "subject", AuthorName: "Name", Body: string(make([]byte, 4097)), CreatedAt: time.Now()},
	} {
		if _, err := store.createMessage(context.Background(), value); err == nil {
			t.Errorf("createMessage(%#v) succeeded", value)
		}
	}
	created, err := store.createMessage(context.Background(), message{
		AuthorSubject: "subject", AuthorName: "Name", Body: "line one\r\nline two", CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Body != "line one\nline two" {
		t.Fatalf("normalized body = %q", created.Body)
	}
}

func TestMessageCursorCodecRejectsMalformedValues(t *testing.T) {
	stamp := time.Now().UTC().Truncate(time.Microsecond)
	raw, err := encodeMessageCursor(messageCursor{CreatedAt: stamp, ID: 42})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeMessageCursor(raw)
	if err != nil || !decoded.CreatedAt.Equal(stamp) || decoded.ID != 42 {
		t.Fatalf("decoded cursor = %#v, %v", decoded, err)
	}
	for _, malformed := range []string{"%", "YWJj", ""} {
		if malformed == "" {
			continue
		}
		if _, err := decodeMessageCursor(malformed); err == nil {
			t.Errorf("decodeMessageCursor(%q) succeeded", malformed)
		}
	}
}

func TestConcurrentMessageInsertionsHaveDistinctIDs(t *testing.T) {
	ctx := context.Background()
	store, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var group sync.WaitGroup
	ids := make(chan int64, 16)
	errs := make(chan error, 16)
	for index := range 16 {
		group.Add(1)
		go func() {
			defer group.Done()
			created, err := store.createMessage(ctx, message{
				AuthorSubject: "subject", AuthorName: "Name", Body: "message", CreatedAt: time.Now().UTC().Add(time.Duration(index)),
			})
			if err != nil {
				errs <- err
				return
			}
			ids <- created.ID
		}()
	}
	group.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Errorf("insert error: %v", err)
	}
	seen := map[int64]struct{}{}
	for id := range ids {
		if _, duplicate := seen[id]; duplicate {
			t.Errorf("duplicate message id %d", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != 16 {
		t.Fatalf("inserted IDs = %d, want 16", len(seen))
	}
}
