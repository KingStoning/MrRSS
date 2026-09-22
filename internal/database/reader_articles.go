package database

import (
	"context"
	"database/sql"
	"fmt"

	"MrRSS/internal/models"
)

// SaveReaderArticle atomically stores remote identity, metadata and the source
// body. A remote ID is stable even when multiple entries share a title/date.
// Existing local reading state and full-text content are preserved.
func (db *DB) SaveReaderArticle(ctx context.Context, provider string, a *models.Article, content string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM articles WHERE feed_id=? AND
		(freshrss_item_id=? OR (url<>'' AND url=?))
		ORDER BY freshrss_item_id=? DESC, id LIMIT 1`, a.FeedID, a.FreshRSSItemID, a.URL, a.FreshRSSItemID).Scan(&id)
	if err == sql.ErrNoRows {
		result, e := tx.ExecContext(ctx, `INSERT INTO articles
		(feed_id,title,url,image_url,published_at,is_read,is_favorite,freshrss_item_id,unique_id)
		VALUES (?,?,?,?,?,?,?,?,?)`, a.FeedID, a.Title, a.URL, a.ImageURL, a.PublishedAt, a.IsRead, a.IsFavorite, a.FreshRSSItemID,
			fmt.Sprintf("reader:%s:%d:%s", provider, a.FeedID, a.FreshRSSItemID))
		if e != nil {
			return e
		}
		id, err = result.LastInsertId()
	} else if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE articles SET freshrss_item_id=?,title=?,url=?,image_url=CASE WHEN image_url='' THEN ? ELSE image_url END WHERE id=?`, a.FreshRSSItemID, a.Title, a.URL, a.ImageURL, id)
	}
	if err != nil {
		return err
	}
	// Also repair bodies missed by older sync versions. Do not overwrite an
	// existing extracted full-text body with a shorter source summary.
	_, err = tx.ExecContext(ctx, `INSERT INTO article_contents(article_id,content,fetched_at) VALUES(?,?,CURRENT_TIMESTAMP)
		ON CONFLICT(article_id) DO UPDATE SET content=excluded.content,fetched_at=excluded.fetched_at WHERE trim(article_contents.content)=''`, id, content)
	if err != nil {
		return err
	}
	return tx.Commit()
}
