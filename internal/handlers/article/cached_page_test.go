package article_test

import (
	"MrRSS/internal/handlers/article"
	"MrRSS/internal/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestArticlePageIncludesPersistedBodiesWithoutNetwork(t *testing.T) {
	h := setupHandler(t)
	defer h.DB.Close()
	id, _ := h.DB.AddFeed(&models.Feed{Title: "Offline", URL: "https://invalid.example/rss"})
	h.DB.SaveArticle(&models.Article{FeedID: id, Title: "Offline article", URL: "https://invalid.example/1"})
	a, _ := h.DB.GetArticleByURL("https://invalid.example/1")
	h.DB.SetArticleContent(a.ID, "<p>available offline</p>")
	w := httptest.NewRecorder()
	article.HandleArticles(h, w, httptest.NewRequest(http.MethodGet, "/api/articles?include_content=true", nil))
	var got []models.Article
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].CachedContent == nil || *got[0].CachedContent != "<p>available offline</p>" {
		t.Fatal("page lacks persisted body")
	}
}
