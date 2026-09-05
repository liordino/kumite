package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// The thinking-tag literals, composed from parts so the source cannot be
// confused with anything else. Matches prompts/agent_template.md and
// ARCHITECTURE.md exactly.
var (
	ThinkOpen  = "\x3c" + "th" + "ink" + "\x3e"
	ThinkClose = "\x3c" + "/th" + "ink" + "\x3e"
)

// MockRequest exposes the parsed request for route matchers.
type MockRequest struct {
	System   string
	Messages []MockMessage
	Last     string // content of the last user message
	Stream   bool
	Format   json.RawMessage
}

type MockMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// RecordedCall is one request the mock received.
type RecordedCall struct {
	System    string
	Last      string
	Messages  []MockMessage
	Stream    bool
	HasFormat bool
}

type mockRoute struct {
	match   func(MockRequest) bool
	respond func(MockRequest) string
}

// MockLLM is a local HTTP server standing in for the OpenAI-compatible
// provider. Tests register routes (first match wins) and can assert on the
// recorded calls. Mock infrastructure is built before the execution engine.
type MockLLM struct {
	Server *httptest.Server

	mu           sync.Mutex
	routes       []mockRoute
	calls        []RecordedCall
	rejectFormat bool // true → 400 on any request carrying a format schema
}

func NewMockLLM() *MockLLM {
	m := &MockLLM{}
	m.Server = httptest.NewServer(http.HandlerFunc(m.serve))
	return m
}

// Register adds a route; the first matcher that returns true handles the call.
func (m *MockLLM) Register(match func(MockRequest) bool, respond func(MockRequest) string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.routes = append(m.routes, mockRoute{match: match, respond: respond})
}

// SetRejectFormat simulates a provider that rejects the format parameter.
func (m *MockLLM) SetRejectFormat(b bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rejectFormat = b
}

func (m *MockLLM) Calls() []RecordedCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]RecordedCall, len(m.calls))
	copy(out, m.calls)
	return out
}

func (m *MockLLM) CallCount(match func(RecordedCall) bool) int {
	n := 0
	for _, c := range m.Calls() {
		if match(c) {
			n++
		}
	}
	return n
}

// Always matches every request.
func Always(MockRequest) bool { return true }

// ContentContains matches when system or any message contains substr.
func ContentContains(substr string) func(MockRequest) bool {
	return func(req MockRequest) bool {
		if strings.Contains(req.System, substr) {
			return true
		}
		for _, msg := range req.Messages {
			if strings.Contains(msg.Content, substr) {
				return true
			}
		}
		return false
	}
}

func (m *MockLLM) serve(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	var raw struct {
		Model    string          `json:"model"`
		Messages []MockMessage   `json:"messages"`
		Stream   bool            `json:"stream"`
		Format   json.RawMessage `json:"format"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	req := MockRequest{Messages: raw.Messages, Stream: raw.Stream, Format: raw.Format}
	for _, msg := range raw.Messages {
		if msg.Role == "system" {
			req.System = msg.Content
		}
		if msg.Role == "user" {
			req.Last = msg.Content
		}
	}

	m.mu.Lock()
	m.calls = append(m.calls, RecordedCall{
		System: req.System, Last: req.Last, Messages: req.Messages,
		Stream: req.Stream, HasFormat: len(raw.Format) > 0,
	})
	reject := m.rejectFormat
	routes := make([]mockRoute, len(m.routes))
	copy(routes, m.routes)
	m.mu.Unlock()

	if reject && len(raw.Format) > 0 {
		http.Error(w, `{"error":{"message":"format parameter not supported"}}`, http.StatusBadRequest)
		return
	}

	for _, route := range routes {
		if route.match(req) {
			content := route.respond(req)
			if raw.Stream {
				m.writeSSE(w, content)
			} else {
				m.writeJSON(w, content)
			}
			return
		}
	}
	http.Error(w, "no fixture registered for prompt", http.StatusInternalServerError)
}

func (m *MockLLM) writeJSON(w http.ResponseWriter, content string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"choices": []map[string]any{
			{"message": map[string]any{"content": content}},
		},
	})
}

// writeSSE streams the content in small chunks as OpenAI-style deltas.
func (m *MockLLM) writeSSE(w http.ResponseWriter, content string) {
	w.Header().Set("Content-Type", "text/event-stream")
	flusher, _ := w.(http.Flusher)
	for _, chunk := range chunkString(content, 32) {
		payload, _ := json.Marshal(map[string]any{
			"choices": []map[string]any{
				{"delta": map[string]any{"content": chunk}},
			},
		})
		fmt.Fprintf(w, "data: %s\n\n", payload)
		if flusher != nil {
			flusher.Flush()
		}
	}
	fmt.Fprint(w, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}

func chunkString(s string, size int) []string {
	var out []string
	for len(s) > size {
		out = append(out, s[:size])
		s = s[size:]
	}
	if len(s) > 0 {
		out = append(out, s)
	}
	return out
}
