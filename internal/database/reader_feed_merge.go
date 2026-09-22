package database

import (
	"context"
)

// ConsolidateReaderFeeds links an ordinary local subscription to the identical
// remote subscription. Keep the local feed ID/settings and every unique article;
// coalesce duplicate URLs while moving their content, conversations and queue.
// Other providers and custom/script feeds are deliberately not combined.
func (db *DB) ConsolidateReaderFeeds(ctx context.Context, provider string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT l.id,r.id,r.freshrss_stream_id FROM feeds l JOIN feeds r ON l.url=r.url
		WHERE l.is_freshrss_source=0 AND r.is_freshrss_source=1 AND r.sync_provider=?
		AND COALESCE(l.script_path,'')='' AND COALESCE(l.type,'') IN ('','rss')`, provider)
	if err != nil {
		return err
	}
	type pair struct {
		local, remote int64
		stream        string
	}
	pairs := []pair{}
	for rows.Next() {
		var p pair
		if err = rows.Scan(&p.local, &p.remote, &p.stream); err != nil {
			rows.Close()
			return err
		}
		pairs = append(pairs, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, p := range pairs {
		dups, err := tx.QueryContext(ctx, `SELECT l.id,r.id,l.is_read,r.is_read,l.is_favorite,r.is_favorite,l.url FROM articles l JOIN articles r ON l.url=r.url WHERE l.feed_id=? AND r.feed_id=? AND l.url<>''`, p.local, p.remote)
		if err != nil {
			return err
		}
		type duplicate struct {
			local, remote  int64
			lr, rr, ls, rs bool
			url            string
		}
		items := []duplicate{}
		for dups.Next() {
			var d duplicate
			if err = dups.Scan(&d.local, &d.remote, &d.lr, &d.rr, &d.ls, &d.rs, &d.url); err != nil {
				dups.Close()
				return err
			}
			items = append(items, d)
		}
		err = dups.Err()
		dups.Close()
		if err != nil {
			return err
		}
		for _, d := range items {
			for _, q := range []string{
				`INSERT INTO article_contents(article_id,content,fetched_at) SELECT ?,content,fetched_at FROM article_contents WHERE article_id=? ON CONFLICT(article_id) DO UPDATE SET content=excluded.content WHERE trim(article_contents.content)=''`,
				`UPDATE chat_sessions SET article_id=? WHERE article_id=?`,
				`UPDATE freshrss_sync_queue SET article_id=? WHERE article_id=?`,
			} {
				if _, err = tx.ExecContext(ctx, q, d.local, d.remote); err != nil {
					return err
				}
			}
			_, err = tx.ExecContext(ctx, `UPDATE articles SET
				freshrss_item_id=(SELECT freshrss_item_id FROM articles WHERE id=?),
				is_read=is_read OR ?,is_favorite=is_favorite OR ?,
				is_read_later=is_read_later OR (SELECT is_read_later FROM articles WHERE id=?),
				is_hidden=is_hidden OR (SELECT is_hidden FROM articles WHERE id=?),
                summary=CASE WHEN COALESCE(summary,'')='' THEN (SELECT summary FROM articles WHERE id=?) ELSE summary END,
                original_summary=CASE WHEN COALESCE(original_summary,'')='' THEN (SELECT original_summary FROM articles WHERE id=?) ELSE original_summary END,
                translated_title=CASE WHEN COALESCE(translated_title,'')='' THEN (SELECT translated_title FROM articles WHERE id=?) ELSE translated_title END,
                image_url=CASE WHEN COALESCE(image_url,'')='' THEN (SELECT image_url FROM articles WHERE id=?) ELSE image_url END,
                audio_url=CASE WHEN COALESCE(audio_url,'')='' THEN (SELECT audio_url FROM articles WHERE id=?) ELSE audio_url END,
                video_url=CASE WHEN COALESCE(video_url,'')='' THEN (SELECT video_url FROM articles WHERE id=?) ELSE video_url END,
                author=CASE WHEN COALESCE(author,'')='' THEN (SELECT author FROM articles WHERE id=?) ELSE author END
                WHERE id=?`, d.remote, d.rr, d.rs, d.remote, d.remote, d.remote, d.remote, d.remote, d.remote, d.remote, d.remote, d.remote, d.local)
			if err != nil {
				return err
			}
			// Preserve positive local actions when first linking an existing feed.
			for _, a := range []struct {
				needed bool
				action string
			}{{d.lr && !d.rr, "mark_read"}, {d.ls && !d.rs, "star"}} {
				if a.needed {
					if _, err = tx.ExecContext(ctx, `INSERT INTO freshrss_sync_queue(article_id,article_url,sync_action,created_at) VALUES(?,?,?,strftime('%s','now'))`, d.local, d.url, a.action); err != nil {
						return err
					}
				}
			}
			if _, err = tx.ExecContext(ctx, `DELETE FROM articles WHERE id=?`, d.remote); err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, `UPDATE articles SET feed_id=? WHERE feed_id=?`, p.local, p.remote); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO feed_tags(feed_id,tag_id) SELECT ?,tag_id FROM feed_tags WHERE feed_id=?`, p.local, p.remote); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO feed_content_options(feed_id,content_selector,remove_selector,cookie,cookie_origin) SELECT ?,content_selector,remove_selector,cookie,cookie_origin FROM feed_content_options WHERE feed_id=?`, p.local, p.remote); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM feeds WHERE id=?`, p.remote); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE feeds SET is_freshrss_source=1,sync_provider=?,freshrss_stream_id=? WHERE id=?`, provider, p.stream, p.local); err != nil {
			return err
		}
	}
	return tx.Commit()
}
