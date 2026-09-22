package database

import (
	"MrRSS/internal/models"
	"context"
	"testing"
	"time"
)

func TestReaderBodiesIdentityRepairAndAtomicity(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.Init(); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	feedID, err := db.AddFeed(&models.Feed{Title: "Reader", URL: "https://example.org/rss", IsFreshRSSSource: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, remote := range []string{"1", "2"} {
		a := &models.Article{FeedID: feedID, Title: "Same recurring title", URL: "https://example.org/" + remote, FreshRSSItemID: remote, PublishedAt: time.Now()}
		if err = db.SaveReaderArticle(ctx, "freshrss", a, "<p>body "+remote+"</p>"); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	db.QueryRow("SELECT count(*) FROM articles").Scan(&count)
	if count != 2 {
		t.Fatalf("same-title entries collapsed: %d", count)
	}
	a, _ := db.GetArticleByURL("https://example.org/1", "freshrss")
	db.SetArticleFavorite(a.ID, true)
	db.DeleteArticleContent(a.ID)
	if err = db.SaveReaderArticle(ctx, "freshrss", &models.Article{FeedID: feedID, Title: "Edited title", URL: a.URL, FreshRSSItemID: "1"}, "<p>repaired</p>"); err != nil {
		t.Fatal(err)
	}
	got, _ := db.GetArticleByID(a.ID)
	body, found, _ := db.GetArticleContent(a.ID)
	if !got.IsFavorite || !found || body != "<p>repaired</p>" {
		t.Fatal("repair lost local state/body")
	}
	db.Exec(`CREATE TRIGGER reject_body BEFORE INSERT ON article_contents BEGIN SELECT RAISE(ABORT,'disk full'); END`)
	err = db.SaveReaderArticle(ctx, "freshrss", &models.Article{FeedID: feedID, Title: "Third", URL: "https://example.org/3", FreshRSSItemID: "3"}, "body")
	if err == nil {
		t.Fatal("expected storage failure")
	}
	db.QueryRow("SELECT count(*) FROM articles").Scan(&count)
	if count != 2 {
		t.Fatal("metadata committed without body")
	}
}

func TestConsolidateReaderFeedsPreservesLocalData(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.Init(); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	local, _ := db.AddFeed(&models.Feed{Title: "My name", URL: "https://example.org/rss", Category: "My folder"})
	remote, _ := db.AddFeed(&models.Feed{Title: "Remote", URL: "https://example.org/rss", IsFreshRSSSource: true, FreshRSSStreamID: "feed/1"})
	other, _ := db.AddFeed(&models.Feed{Title: "Other provider", URL: "https://example.org/rss", IsFreshRSSSource: true, SyncProvider: "miniflux"})
	db.SaveArticle(&models.Article{FeedID: local, Title: "A", URL: "https://example.org/1", IsFavorite: true, IsReadLater: true})
	l, _ := db.GetArticleByURL("https://example.org/1")
	if err = db.SaveReaderArticle(ctx, "freshrss", &models.Article{FeedID: remote, Title: "A", URL: l.URL, FreshRSSItemID: "remote-1"}, "saved body"); err != nil {
		t.Fatal(err)
	}
	r, _ := db.GetArticleByURL(l.URL, "freshrss")
	db.Exec(`INSERT INTO chat_sessions(article_id,title) VALUES(?,'keep conversation')`, r.ID)
	db.EnqueueSyncChange(r.ID, r.URL, SyncActionMarkUnread)
	for i := 0; i < 2; i++ {
		if err = db.ConsolidateReaderFeeds(ctx, "freshrss"); err != nil {
			t.Fatal(err)
		}
	}
	feeds, _ := db.GetFeeds()
	if len(feeds) != 2 {
		t.Fatalf("duplicate feed remains: %d", len(feeds))
	}
	f, _ := db.GetFeedByID(local)
	if !f.IsFreshRSSSource || f.Title != "My name" || f.Category != "My folder" {
		t.Fatal("local feed identity/settings lost")
	}
	if _, err = db.GetFeedByID(other); err != nil {
		t.Fatal("other provider changed")
	}
	a, _ := db.GetArticleByID(l.ID)
	if !a.IsFavorite || !a.IsReadLater || a.FreshRSSItemID != "remote-1" {
		t.Fatal("article state lost")
	}
	body, _, _ := db.GetArticleContent(l.ID)
	if body != "saved body" {
		t.Fatal("body lost")
	}
	var count int
	db.QueryRow("SELECT count(*) FROM articles").Scan(&count)
	if count != 1 {
		t.Fatal("duplicate article remains")
	}
	db.QueryRow("SELECT count(*) FROM chat_sessions WHERE article_id=?", l.ID).Scan(&count)
	if count != 1 {
		t.Fatal("conversation lost")
	}
	pending, _ := db.GetPendingSyncChanges(-1, "freshrss")
	if len(pending) != 2 {
		t.Fatalf("pending actions lost: %d", len(pending))
	}
}
