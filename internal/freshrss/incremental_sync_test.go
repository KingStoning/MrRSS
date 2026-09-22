package freshrss

import (
	"MrRSS/internal/database"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIncrementalSyncSkipsCachedBodiesAndReconcilesBothDirections(t *testing.T) {
	db, err := database.NewDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.Init(); err != nil {
		t.Fatal(err)
	}
	bodies, unread, starred, fail := 0, true, true, false
	failEdit := false
	editActions := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/token"):
			fmt.Fprint(w, "test")
		case strings.HasSuffix(r.URL.Path, "/edit-tag"):
			_ = r.ParseForm()
			if failEdit && r.Form.Get("a") == TagStarred {
				http.Error(w, "retry", 503)
				return
			}
			editActions = append(editActions, r.Form.Get("a")+"/"+r.Form.Get("r"))
			fmt.Fprint(w, "OK")
		case strings.HasSuffix(r.URL.Path, "ClientLogin"):
			fmt.Fprint(w, "Auth=test\n")
		case strings.HasSuffix(r.URL.Path, "subscription/list"):
			fmt.Fprint(w, `{"subscriptions":[{"id":"feed/1","url":"https://example.org/rss","title":"News"}]}`)
		case strings.HasSuffix(r.URL.Path, "stream/items/ids"):
			if fail {
				http.Error(w, "offline", 503)
				return
			}
			hasItem := true
			if r.URL.Query().Get("xt") == TagRead {
				hasItem = unread
			}
			if r.URL.Query().Get("s") == TagStarred {
				hasItem = starred
			}
			if hasItem {
				fmt.Fprint(w, `{"itemRefs":[{"id":"42"}]}`)
			} else {
				fmt.Fprint(w, `{"itemRefs":[]}`)
			}
		case strings.HasSuffix(r.URL.Path, "stream/items/contents"):
			bodies++
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if r.Form.Get("i") != normalizedItemID("42") {
				t.Error("wrong article identifier")
			}
			fmt.Fprint(w, `{"items":[{"id":"tag:google.com,2005:reader/item/000000000000002a","title":"Article","canonical":[{"href":"https://example.org/42"}],"summary":{"content":"<p>Body</p>"},"published":1700000000,"categories":[],"origin":{"streamId":"feed/1"}}]}`)
		default:
			t.Errorf("unexpected request %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	service := NewBidirectionalSyncService(server.URL, "user", "password", db)
	for i := 0; i < 3; i++ {
		if i == 1 {
			unread = false
			starred = false
		}
		if i == 2 {
			unread = true
			starred = true
		}
		result, err := service.Sync(context.Background())
		if err != nil || !result.PullSuccess {
			t.Fatalf("sync: %+v %v", result, err)
		}
		a, err := db.GetArticleByURL("https://example.org/42", "freshrss")
		if err != nil {
			t.Fatal(err)
		}
		if a.IsRead == unread || a.IsFavorite != starred {
			t.Fatalf("incorrect state read=%v starred=%v", a.IsRead, a.IsFavorite)
		}
	}
	if bodies != 1 {
		t.Fatalf("body downloaded %d times; want once", bodies)
	}
	fail = true
	if _, err := service.Sync(context.Background()); err == nil {
		t.Fatal("failed snapshot reported success")
	}
	a, _ := db.GetArticleByURL("https://example.org/42", "freshrss")
	if a.IsRead || !a.IsFavorite {
		t.Fatal("failed snapshot changed local state")
	}
	if err := db.EnqueueSyncChange(a.ID, a.URL, database.SyncActionUnstar); err != nil {
		t.Fatal(err)
	}
	if err := db.SetArticleFavorite(a.ID, false); err != nil {
		t.Fatal(err)
	}
	id := normalizedItemID("42")
	if _, err := service.applyIDSnapshot(context.Background(), []string{id}, []string{id}, []string{id}); err != nil {
		t.Fatal(err)
	}
	a, _ = db.GetArticleByURL(a.URL, "freshrss")
	if a.IsFavorite {
		t.Fatal("pending local unstar was overwritten")
	}

	baseline, err := service.localStates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.MarkArticleRead(a.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := service.applyIDSnapshot(context.Background(), []string{id}, []string{id}, []string{id}, baseline); err != nil {
		t.Fatal(err)
	}
	a, _ = db.GetArticleByURL(a.URL, "freshrss")
	if !a.IsRead {
		t.Fatal("edit made during network request overwritten")
	}
	if err := db.EnqueueSyncChange(a.ID, a.URL, database.SyncActionStar); err != nil {
		t.Fatal(err)
	}
	if err := db.EnqueueSyncChange(a.ID, a.URL, database.SyncActionMarkRead); err != nil {
		t.Fatal(err)
	}
	pending, err := db.GetPendingSyncChanges(-1, "freshrss")
	if err != nil {
		t.Fatal(err)
	}
	failEdit = true
	if _, err := service.replayPending(context.Background(), pending); err == nil {
		t.Fatal("expected partial push failure")
	}
	stillPending, _ := db.GetPendingSyncChanges(-1, "freshrss")
	if len(stillPending) != len(pending) {
		t.Fatal("partial push acknowledged only the newest intent")
	}
	failEdit = false
	editActions = nil
	if _, err := service.replayPending(context.Background(), stillPending); err != nil {
		t.Fatal(err)
	}
	for _, action := range editActions {
		if action == "/"+TagStarred {
			t.Fatal("older unstar overrode the newest star")
		}
	}
	remaining, _ := db.GetPendingSyncCount("freshrss")
	if remaining != 0 {
		t.Fatalf("queue not acknowledged: %d", remaining)
	}
}

func TestStreamIDsRejectsRepeatedContinuation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"itemRefs":[{"id":"42"}],"continuation":"same"}`)
	}))
	defer server.Close()
	c := NewClient(server.URL, "", "")
	c.authToken = "test"
	if _, err := c.streamIDs(context.Background(), "stream", ""); err == nil {
		t.Fatal("repeated page should fail")
	}
}
