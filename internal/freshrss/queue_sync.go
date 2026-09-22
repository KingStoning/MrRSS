package freshrss

import (
	"MrRSS/internal/database"
	"context"
	"fmt"
	"time"
)

// replayPending acknowledges the entire captured queue only after every request
// succeeds. This prevents an older offline edit from resurfacing after a partial
// failure while leaving edits created during the network requests untouched.
func (s *BidirectionalSyncService) replayPending(ctx context.Context, pending []database.SyncQueueItem) (int, error) {
	latest := map[string]database.SyncQueueItem{}
	for _, item := range pending {
		field := "read"
		if item.Action == database.SyncActionStar || item.Action == database.SyncActionUnstar {
			field = "star"
		}
		key := fmt.Sprintf("%d/%s", item.ArticleID, field)
		if previous, ok := latest[key]; !ok || previous.ID < item.ID {
			latest[key] = item
		}
	}
	groups := map[database.SyncAction][]string{}
	for _, item := range latest {
		article, err := s.db.GetArticleByID(item.ArticleID)
		if err != nil {
			return 0, err
		}
		if article.FreshRSSItemID == "" {
			return 0, fmt.Errorf("article %d has no remote item ID", item.ArticleID)
		}
		groups[item.Action] = append(groups[item.Action], article.FreshRSSItemID)
	}
	count := 0
	for _, op := range []struct {
		action database.SyncAction
		send   func(context.Context, []string) error
	}{
		{database.SyncActionMarkRead, s.client.MarkAsReadBatch},
		{database.SyncActionMarkUnread, s.client.MarkAsUnreadBatch},
		{database.SyncActionStar, s.client.StarBatch},
		{database.SyncActionUnstar, s.client.UnstarBatch},
	} {
		ids := groups[op.action]
		for start := 0; start < len(ids); start += 100 {
			batch := ids[start:min(start+100, len(ids))]
			if err := op.send(ctx, batch); err != nil {
				return count, err
			}
			count += len(batch)
		}
	}
	if len(pending) == 0 {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return count, err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, "UPDATE freshrss_sync_queue SET synced_at=?,sync_error=NULL WHERE id=?")
	if err != nil {
		return count, err
	}
	defer stmt.Close()
	for _, item := range pending {
		if _, err = stmt.ExecContext(ctx, time.Now().Unix(), item.ID); err != nil {
			return count, err
		}
	}
	return count, tx.Commit()
}
