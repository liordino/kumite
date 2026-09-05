package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
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

// RepoAgent is one specialist prompt file discovered in the upstream repo
// (advanced mode — the full roster beyond the curated thirteen).
type RepoAgent struct {
	AgentID     string `json:"agent_id"`
	DisplayName string `json:"display_name"`
	SourceURL   string `json:"source_url"`
}

// RepoTreeURL derives the GitHub tree API URL from the raw content base
// (https://raw.githubusercontent.com/<owner>/<repo>/<ref>).
func RepoTreeURL(rawBase string) (string, bool) {
	const prefix = "https://raw.githubusercontent.com/"
	if !strings.HasPrefix(rawBase, prefix) {
		return "", false
	}
	rest := strings.Trim(strings.TrimPrefix(rawBase, prefix), "/")
	parts := strings.Split(rest, "/")
	if len(parts) < 3 {
		return "", false
	}
	owner, repo, ref := parts[0], parts[1], parts[2]
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", owner, repo, ref), true
}

// repoCacheID is the agent_cache pseudo-entry that stores the repo listing.
const repoCacheID = "__repo_tree__"

// FetchRepoListing lists every specialist prompt file in the upstream repo
// via the GitHub tree API, cached like any other fetch: fresh within the
// TTL, revalidated after, served stale when GitHub is unreachable.
func FetchRepoListing(ctx context.Context, d *db.DB, client *http.Client, treeAPIURL, rawBase string, ttl time.Duration) ([]RepoAgent, error) {
	row, rowErr := d.GetAgentCache(repoCacheID)
	if rowErr == nil {
		if fetchedAt, parseErr := time.Parse(time.RFC3339, row.FetchedAt); parseErr == nil && time.Since(fetchedAt) < ttl {
			return decodeRepoListing(row.Content)
		}
	}

	listing, err := fetchRepoTree(ctx, client, treeAPIURL, rawBase)
	if err == nil {
		if encoded, mErr := json.Marshal(listing); mErr == nil {
			if upErr := d.UpsertAgentCache(repoCacheID, treeAPIURL, string(encoded), ""); upErr != nil {
				return nil, fmt.Errorf("repo listing cache: %w", upErr)
			}
		}
		return listing, nil
	}
	if rowErr == nil && row.Content != "" {
		return decodeRepoListing(row.Content) // stale
	}
	return nil, fmt.Errorf("repo listing: %w", err)
}

func fetchRepoTree(ctx context.Context, client *http.Client, treeAPIURL, rawBase string) ([]RepoAgent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, treeAPIURL, nil)
	if err != nil {
		return nil, &UnreachableError{Err: err}
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, &UnreachableError{Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, &HTTPError{Status: resp.StatusCode, Body: string(b)}
	}
	var tree struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&tree); err != nil {
		return nil, fmt.Errorf("repo tree decode: %w", err)
	}

	listing := make([]RepoAgent, 0, 32)
	for _, entry := range tree.Tree {
		if entry.Type != "blob" || !strings.HasSuffix(entry.Path, ".md") {
			continue
		}
		if strings.EqualFold(entry.Path, "README.md") {
			continue
		}
		base := path.Base(entry.Path)
		id := strings.TrimSuffix(base, ".md")
		listing = append(listing, RepoAgent{
			AgentID:     id,
			DisplayName: displayNameFromID(id),
			SourceURL:   strings.TrimRight(rawBase, "/") + "/" + entry.Path,
		})
	}
	if len(listing) == 0 {
		return nil, fmt.Errorf("repo tree contained no agent prompt files")
	}
	return listing, nil
}

func decodeRepoListing(raw string) ([]RepoAgent, error) {
	var out []RepoAgent
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("decode cached repo listing: %w", err)
	}
	return out, nil
}

// displayNameFromID turns "product-product-manager" into "Product Product
// Manager" — mechanical, never specialist judgment.
func displayNameFromID(id string) string {
	words := strings.Split(strings.ReplaceAll(id, "-", " "), " ")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
