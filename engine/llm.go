package engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"kumite/models"
)

// engine/llm.go is the ONLY place that talks to the inference provider. No
// other package makes HTTP requests to it.

const correctionPrompt = `Your previous response was not valid JSON matching the required schema.
Here is the schema again: %[1]s.
Respond with only valid JSON. No preamble, no markdown fences.`

const maxCorrectionRetries = 2

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is one inference request. Format carries the JSON schema for
// the `format` parameter (nil = unconstrained markdown call — the Uchikomi
// and Phase 2 exception).
type ChatRequest struct {
	System   string
	Messages []Message
	Format   map[string]any
	Stream   bool
}

// RuntimeLLMConfig is resolved from app_config at call time so provider
// changes take effect without restart.
type RuntimeLLMConfig struct {
	Endpoint string
	Model    string
	APIKey   string
}

// Result is the outcome of one structured-output call.
type Result struct {
	Content  string // final text, thinking stripped
	Thinking string // extracted <think> block content, "" when the model emitted none
	Partial  bool   // true when content came from the markdown-extraction last resort
}

// Hooks receives streamed deltas. OnThinking fires for <think> content as it
// arrives; OnChunk for answer content. Both are optional.
type Hooks struct {
	OnThinking func(chunk string)
	OnChunk    func(delta string)
}

// HTTPError distinguishes provider HTTP failures (used for 502 mapping).
type HTTPError struct {
	Status int
	Body   string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("llm endpoint returned status %d: %s", e.Status, truncate(e.Body, 300))
}

// UnreachableError wraps transport failures (connection refused, DNS,
// timeout) — mapped to 502 upstream.
type UnreachableError struct{ Err error }

func (e *UnreachableError) Error() string {
	return fmt.Sprintf("llm endpoint unreachable: %v", e.Err)
}

func (e *UnreachableError) Unwrap() error { return e.Err }

type Client struct {
	HTTP *http.Client
}

func NewClient() *Client {
	return &Client{HTTP: &http.Client{Timeout: 10 * time.Minute}}
}

// JSON runs the full structured-output strategy against an endpoint and
// unmarshals the response into target:
//
//	Layer 1: `format` schema constraint (primary path)
//	Layer 2: retry with correction message (max 2 attempts) — also the
//	         compatibility floor when the provider rejects `format`
//	Layer 3: permissive markdown extraction of a JSON object (marks Partial)
//
// When the response unmarshals via Layer 1/2 the result is not Partial. When
// only Layer 3 could produce JSON the result is Partial — the caller stores
// it with status "partial" and never drops it.
func (c *Client) JSON(ctx context.Context, cfg RuntimeLLMConfig, req ChatRequest, target any, hooks Hooks) (Result, error) {
	var lastContent, thinking string

	for attempt := 0; ; attempt++ {
		res, err := c.call(ctx, cfg, req, hooks)
		if err != nil {
			return Result{}, err
		}
		lastContent, thinking = res.Content, res.Thinking
		if err := json.Unmarshal([]byte(lastContent), target); err == nil {
			return Result{Content: lastContent, Thinking: thinking}, nil
		}
		if attempt >= maxCorrectionRetries {
			break
		}
		// Layer 2: retry with correction. The schema is restated each time.
		schemaJSON, _ := json.Marshal(req.Format)
		req.Messages = append(req.Messages,
			Message{Role: "assistant", Content: lastContent},
			Message{Role: "user", Content: fmt.Sprintf(correctionPrompt, string(schemaJSON))},
		)
	}

	// Layer 3: permissive extraction of a JSON object from the raw response.
	extracted, ok := ExtractJSONObject(lastContent)
	if ok {
		if err := json.Unmarshal([]byte(extracted), target); err == nil {
			return Result{Content: extracted, Thinking: thinking, Partial: true}, nil
		}
	}
	return Result{}, fmt.Errorf("llm structured output failed after %d correction attempts: response was not valid JSON", maxCorrectionRetries)
}

// Complete performs one unconstrained call (markdown outputs: Uchikomi,
// Phase 2, handoff). No retry loop — markdown calls are validated
// structurally by their callers, and a failed Phase 2 is retried by
// re-running synthesis, not inside this layer.
func (c *Client) Complete(ctx context.Context, cfg RuntimeLLMConfig, req ChatRequest, hooks Hooks) (Result, error) {
	return c.call(ctx, cfg, req, hooks)
}

// call performs one HTTP round trip, degrading gracefully when the provider
// rejects the `format` parameter.
func (c *Client) call(ctx context.Context, cfg RuntimeLLMConfig, req ChatRequest, hooks Hooks) (Result, error) {
	content, err := c.roundTrip(ctx, cfg, req, hooks)
	if err == nil {
		return SplitThinking(content)
	}
	var httpErr *HTTPError
	if errors.As(err, &httpErr) && req.Format != nil && httpErr.Status >= 400 && httpErr.Status < 500 {
		// The provider does not support `format` — degrade to Layer 2 rather
		// than failing. The retry loop is the compatibility floor.
		req.Format = nil
		content, err = c.roundTrip(ctx, cfg, req, hooks)
		if err != nil {
			return Result{}, err
		}
		return SplitThinking(content)
	}
	return Result{}, err
}

func (c *Client) roundTrip(ctx context.Context, cfg RuntimeLLMConfig, req ChatRequest, hooks Hooks) (string, error) {
	if cfg.Endpoint == "" || cfg.Model == "" {
		return "", fmt.Errorf("llm config incomplete: endpoint and model must be set")
	}
	url := joinEndpoint(cfg.Endpoint)

	messages := make([]Message, 0, len(req.Messages)+1)
	if req.System != "" {
		messages = append(messages, Message{Role: "system", Content: req.System})
	}
	messages = append(messages, req.Messages...)

	body := map[string]any{
		"model":    cfg.Model,
		"messages": messages,
		"stream":   req.Stream,
	}
	if req.Format != nil {
		body["format"] = req.Format
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("llm encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", &UnreachableError{Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return "", &UnreachableError{Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", &HTTPError{Status: resp.StatusCode, Body: string(b)}
	}

	if req.Stream {
		return readStream(resp.Body, hooks)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return "", fmt.Errorf("llm read response: %w", err)
	}
	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &completion); err != nil {
		return "", fmt.Errorf("llm decode response: %w", err)
	}
	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("llm response contained no choices")
	}
	return completion.Choices[0].Message.Content, nil
}

// readStream consumes an OpenAI-compatible SSE stream. Content deltas go to
// hooks.OnChunk; thinking is separated by the incremental splitter below.
func readStream(r io.Reader, hooks Hooks) (string, error) {
	var sb strings.Builder
	splitter := &thinkSplitter{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue // tolerate keep-alive comments and partial lines
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		sb.WriteString(delta)
		contentDelta, thinkingDelta := splitter.feed(delta)
		if thinkingDelta != "" && hooks.OnThinking != nil {
			hooks.OnThinking(thinkingDelta)
		}
		if contentDelta != "" && hooks.OnChunk != nil {
			hooks.OnChunk(contentDelta)
		}
	}
	if err := sc.Err(); err != nil {
		return sb.String(), fmt.Errorf("llm stream read: %w", err)
	}
	if tail := splitter.finish(); tail != "" && hooks.OnChunk != nil {
		hooks.OnChunk(tail)
	}
	return sb.String(), nil
}

// joinEndpoint appends the chat completions path unless the endpoint already
// points at it.
func joinEndpoint(endpoint string) string {
	e := strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(e, "/chat/completions") {
		return e
	}
	return e + "/chat/completions"
}

// --- thinking trace handling ---

var thinkOpenRe = regexp.MustCompile(`<think>`)

// splitThinking extracts a leading (or embedded) <think>...</think> block
// from content. Models that emit no reasoning blocks yield an empty Thinking.
// An unterminated block is treated as all-thinking with empty content — the
// response carries nothing else.
func SplitThinking(content string) (Result, error) {
	idx := strings.Index(content, "<think>")
	if idx < 0 {
		return Result{Content: content, Thinking: ""}, nil
	}
	rest := content[idx+len("<think>"):]
	end := strings.Index(rest, "</think>")
	if end < 0 {
		return Result{Content: "", Thinking: strings.TrimSpace(rest)}, nil
	}
	thinking := strings.TrimSpace(rest[:end])
	answer := content[:idx] + rest[end+len("</think>"):]
	return Result{Content: strings.TrimSpace(answer), Thinking: thinking}, nil
}

// thinkSplitter separates thinking from content incrementally in a stream.
type thinkSplitter struct {
	buf        strings.Builder // holds unconsumed text (may contain a partial tag)
	inThinking bool
	done       bool
}

func (t *thinkSplitter) feed(delta string) (contentDelta, thinkingDelta string) {
	if t.done {
		return delta, ""
	}
	t.buf.WriteString(delta)
	s := t.buf.String()

	for {
		if !t.inThinking {
			idx := strings.Index(s, "<think>")
			if idx < 0 {
				// Withhold a suffix that could be the start of "<think>".
				keep := partialSuffixLen(s, "<think>")
				emit := s[:len(s)-keep]
				t.buf.Reset()
				t.buf.WriteString(s[len(s)-keep:])
				return emit, ""
			}
			if idx > 0 {
				t.buf.Reset()
				t.buf.WriteString(s[idx:])
				s = s[idx:]
			}
			t.inThinking = true
			s = s[len("<think>"):]
			continue
		}
		end := strings.Index(s, "</think>")
		if end < 0 {
			keep := partialSuffixLen(s, "</think>")
			emit := s[:len(s)-keep]
			t.buf.Reset()
			t.buf.WriteString(s[len(s)-keep:])
			return "", emit
		}
		t.buf.Reset()
		t.inThinking = false
		s = s[end+len("</think>"):]
		if strings.TrimSpace(s) == "" {
			t.done = true
			return "", ""
		}
	}
}

// finish returns remaining content once the stream ends (e.g. an unclosed
// think block means everything buffered was thinking).
func (t *thinkSplitter) finish() string {
	s := t.buf.String()
	t.buf.Reset()
	if t.inThinking || strings.HasPrefix(strings.TrimSpace(s), "<think>") {
		return "" // thinking, not content
	}
	return s
}

// partialSuffixLen returns how many bytes at the end of s are a prefix of tag.
func partialSuffixLen(s, tag string) int {
	max := len(tag) - 1
	if len(s) < max {
		max = len(s)
	}
	for k := max; k > 0; k-- {
		if strings.HasPrefix(tag, s[len(s)-k:]) {
			return k
		}
	}
	return 0
}

// extractJSONObject scans for the first balanced {...} region, respecting
// string literals and escapes. Used by Layer 3 of the structured-output
// strategy.
func ExtractJSONObject(s string) (string, bool) {
	start := strings.Index(s, "{")
	if start < 0 {
		return "", false
	}
	depth := 0
	inStr := false
	esc := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if esc {
			esc = false
			continue
		}
		switch c {
		case '\\':
			if inStr {
				esc = true
			}
		case '"':
			inStr = !inStr
		case '{':
			if !inStr {
				depth++
			}
		case '}':
			if !inStr {
				depth--
				if depth == 0 {
					return s[start : i+1], true
				}
			}
		}
	}
	return "", false
}

// truncate shortens s for error messages.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// MaskAPIKey returns the masked representation used by GET /api/config.
func MaskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	return "••••••••"
}

// SeverityRank orders findings critical-first for rendering and sorting.
func SeverityRank(s models.Severity) int {
	switch s {
	case models.SeverityCritical:
		return 0
	case models.SeverityHigh:
		return 1
	case models.SeverityMedium:
		return 2
	case models.SeverityLow:
		return 3
	}
	return 4
}
