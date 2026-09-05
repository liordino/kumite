package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"kumite/db"
)

// FetchAgentPrompt returns the specialist prompt for agentID, honoring the
// SQLite cache with a TTL and ETag conditional requests:
//
//  1. cache hit and not expired  -> serve cached
//  2. expired                    -> conditional GET with If-None-Match
//  3. 304                        -> reset TTL, serve cached
//  4. 200                        -> update cache, serve fresh
//  5. GitHub unreachable + cache -> serve stale (stale=true)
//
// A failed fetch with an empty cache is an error — nothing to fall back to.
func FetchAgentPrompt(ctx context.Context, d *db.DB, client *http.Client, agentID, sourceURL string, ttl time.Duration) (content string, cached bool, stale bool, err error) {
	row, rowErr := d.GetAgentCache(agentID)
	switch {
	case rowErr == nil:
		fetchedAt, parseErr := time.Parse(time.RFC3339, row.FetchedAt)
		if parseErr == nil && time.Since(fetchedAt) < ttl {
			return row.Content, true, false, nil
		}
	case rowErr == db.ErrNotFound:
		// nothing cached yet — fall through to the network
	default:
		return "", false, false, fmt.Errorf("fetch agent %s: %w", agentID, rowErr)
	}

	conditional := rowErr == nil // only send If-None-Match when we have a stored ETag
	fresh, etag, fetchErr := fetchFromGitHub(ctx, client, sourceURL, row.ETag, conditional)
	if fetchErr == nil {
		if err := d.UpsertAgentCache(agentID, sourceURL, fresh, etag); err != nil {
			return "", false, false, fmt.Errorf("fetch agent %s: %w", agentID, err)
		}
		return fresh, false, false, nil
	}

	var nm *ErrNotModified
	if errors.As(fetchErr, &nm) {
		// 304: reset TTL without re-downloading.
		if touchErr := d.TouchAgentCache(agentID); touchErr != nil {
			return "", false, false, fmt.Errorf("fetch agent %s: %w", agentID, touchErr)
		}
		return row.Content, true, false, nil
	}

	// GitHub unreachable: serve stale when we have it.
	if rowErr == nil && row.Content != "" {
		return row.Content, true, true, nil
	}
	return "", false, false, fmt.Errorf("fetch agent %s: %w", agentID, fetchErr)
}

// ErrNotModified signals a 304 response.
type ErrNotModified struct{}

func (e *ErrNotModified) Error() string { return "not modified" }

func fetchFromGitHub(ctx context.Context, client *http.Client, sourceURL, etag string, conditional bool) (content, newETag string, err error) {
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if reqErr != nil {
		return "", "", &UnreachableError{Err: reqErr}
	}
	if conditional && etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", &UnreachableError{Err: err}
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotModified:
		return "", "", &ErrNotModified{}
	case resp.StatusCode != http.StatusOK:
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", "", &HTTPError{Status: resp.StatusCode, Body: string(b)}
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", "", fmt.Errorf("read agent prompt: %w", err)
	}
	return string(data), resp.Header.Get("ETag"), nil
}
