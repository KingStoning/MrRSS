package freshrss

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Reader IDs use decimal in itemRefs and hexadecimal in stream contents.
func normalizedItemID(id string) string {
	if strings.HasPrefix(id, "tag:google.com,2005:reader/item/") {
		return id
	}
	n, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return id
	}
	return fmt.Sprintf("tag:google.com,2005:reader/item/%016x", n)
}

func (c *Client) streamIDs(ctx context.Context, stream, exclude string) ([]string, error) {
	ids := []string{}
	seen := map[string]bool{}
	continuation := ""
	for {
		params := url.Values{"output": {"json"}, "s": {stream}, "n": {"10000"}}
		if exclude != "" {
			params.Set("xt", exclude)
		}
		if continuation != "" {
			params.Set("c", continuation)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/reader/api/0/stream/items/ids?"+params.Encode(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "GoogleLogin auth="+c.authToken)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		var page struct {
			ItemRefs []struct {
				ID string `json:"id"`
			} `json:"itemRefs"`
			Continuation string `json:"continuation"`
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("item IDs: HTTP %d", resp.StatusCode)
		}
		err = json.NewDecoder(resp.Body).Decode(&page)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		for _, item := range page.ItemRefs {
			ids = append(ids, normalizedItemID(item.ID))
		}
		if page.Continuation == "" {
			return ids, nil
		}
		if seen[page.Continuation] {
			return nil, fmt.Errorf("repeated item ID continuation")
		}
		seen[page.Continuation] = true
		continuation = page.Continuation
	}
}

func (c *Client) itemContents(ctx context.Context, ids []string) ([]Article, error) {
	data := url.Values{"output": {"json"}}
	for _, id := range ids {
		data.Add("i", id)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/reader/api/0/stream/items/contents", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "GoogleLogin auth="+c.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("item contents: HTTP %d", resp.StatusCode)
	}
	page, err := decodeGoogleReaderStreamContents(resp.Body)
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

// syncIncremental transfers lightweight ID snapshots and downloads only missing
// article bodies. All snapshots must succeed before any remote state is applied.
func (s *BidirectionalSyncService) syncIncremental(ctx context.Context) (int, int, error) {
	// Snapshot the queue so edits arriving during this sync remain pending.
	pending, err := s.db.GetPendingSyncChanges(-1, s.provider)
	if err != nil {
		return 0, 0, err
	}
	pushed, err := s.replayPending(ctx, pending)
	if err != nil {
		return 0, pushed, err
	}
	baseline, err := s.localStates(ctx)
	if err != nil {
		return 0, pushed, err
	}
	subscriptions, err := s.client.GetSubscriptions(ctx)
	if err != nil {
		return 0, pushed, fmt.Errorf("subscriptions: %w", err)
	}
	count, err := s.createFeedsFromSubscriptions(ctx, subscriptions)
	if err != nil {
		return count, pushed, err
	}
	const readingList = "user/-/state/com.google/reading-list"
	all, err := s.client.streamIDs(ctx, readingList, "")
	if err != nil {
		return count, pushed, err
	}
	unread, err := s.client.streamIDs(ctx, readingList, TagRead)
	if err != nil {
		return count, pushed, err
	}
	starred, err := s.client.streamIDs(ctx, TagStarred, "")
	if err != nil {
		return count, pushed, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT a.freshrss_item_id FROM articles a JOIN feeds f ON f.id=a.feed_id WHERE f.is_freshrss_source=1 AND f.sync_provider=? AND a.freshrss_item_id != ''`, s.provider)
	if err != nil {
		return count, pushed, err
	}
	known := map[string]bool{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return count, pushed, err
		}
		known[normalizedItemID(id)] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return count, pushed, err
	}
	missing := []string{}
	for _, id := range append(append(all, unread...), starred...) {
		if !known[id] {
			missing = append(missing, id)
			known[id] = true
		}
	}
	for start := 0; start < len(missing); start += 100 {
		end := min(start+100, len(missing))
		articles, err := s.client.itemContents(ctx, missing[start:end])
		if err != nil {
			return count, pushed, err
		}
		saved, err := s.saveArticlesFromServer(ctx, articles)
		if err != nil {
			return count, pushed, err
		}
		count += saved
	}
	changed, err := s.applyIDSnapshot(ctx, all, unread, starred, baseline)
	return count + changed, pushed, err
}

type localState struct{ read, star bool }

func (s *BidirectionalSyncService) localStates(ctx context.Context) (map[int64]localState, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT a.id,a.is_read,a.is_favorite FROM articles a JOIN feeds f ON f.id=a.feed_id WHERE f.is_freshrss_source=1 AND f.sync_provider=?", s.provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	states := map[int64]localState{}
	for rows.Next() {
		var id int64
		var st localState
		if err := rows.Scan(&id, &st.read, &st.star); err != nil {
			return nil, err
		}
		states[id] = st
	}
	return states, rows.Err()
}

func (s *BidirectionalSyncService) applyIDSnapshot(ctx context.Context, all, unread, starred []string, baseline ...map[int64]localState) (int, error) {
	readSet, starSet := map[string]bool{}, map[string]bool{}
	for _, id := range all {
		readSet[id] = true
	}
	for _, id := range unread {
		readSet[id] = false
	}
	for _, id := range starred {
		starSet[id] = true
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT a.id,a.freshrss_item_id,a.is_read,a.is_favorite FROM articles a JOIN feeds f ON f.id=a.feed_id WHERE f.is_freshrss_source=1 AND f.sync_provider=?`, s.provider)
	if err != nil {
		return 0, err
	}
	type state struct {
		id         int64
		remote     string
		read, star bool
	}
	states := []state{}
	for rows.Next() {
		var st state
		if err = rows.Scan(&st.id, &st.remote, &st.read, &st.star); err != nil {
			rows.Close()
			return 0, err
		}
		states = append(states, st)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, st := range states {
		remote := normalizedItemID(st.remote)
		read, exists := readSet[remote]
		if !exists {
			continue
		} // Do not infer state for articles absent from the snapshot.
		for _, field := range []struct {
			column     string
			value, old bool
			a, b       string
		}{
			{"is_read", read, st.read, "mark_read", "mark_unread"}, {"is_favorite", starSet[remote], st.star, "star", "unstar"},
		} {
			if len(baseline) > 0 {
				if before, ok := baseline[0][st.id]; ok {
					if (field.column == "is_read" && before.read != st.read) || (field.column == "is_favorite" && before.star != st.star) {
						continue
					}
				}
			}
			if field.value == field.old {
				continue
			}
			// Only these two constant columns are interpolated; all data uses parameters.
			result, err := tx.ExecContext(ctx, "UPDATE articles SET "+field.column+"=? WHERE id=? AND NOT EXISTS (SELECT 1 FROM freshrss_sync_queue WHERE article_id=? AND synced_at IS NULL AND sync_action IN (?,?))", field.value, st.id, st.id, field.a, field.b)
			if err != nil {
				return changed, err
			}
			n, _ := result.RowsAffected()
			changed += int(n)
		}
	}
	return changed, tx.Commit()
}
