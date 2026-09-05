package tests

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"kumite/engine"
	"kumite/models"
)

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("fixtures/" + name)
	if err != nil {
		t.Fatalf("load fixture %s: %v", name, err)
	}
	return string(b)
}

func runtimeFor(mock *MockLLM) engine.RuntimeLLMConfig {
	return engine.RuntimeLLMConfig{Endpoint: mock.Server.URL, Model: "test-model", APIKey: "test-key"}
}

// TestJSONHappyPath covers Layer 1: the provider constrains generation with
// the format schema and the response parses into the typed struct.
func TestJSONHappyPath(t *testing.T) {
	mock := NewMockLLM()
	defer mock.Server.Close()

	fixture := loadFixture(t, "agent_output.json")
	mock.Register(ContentContains("engineering-software-architect"), func(MockRequest) string { return fixture })

	client := engine.NewClient()
	var out models.AgentOutput
	res, err := client.JSON(context.Background(), runtimeFor(mock), engine.ChatRequest{
		System: "specialist prompt",
		Messages: []engine.Message{
			{Role: "user", Content: "agent_id: engineering-software-architect"},
		},
		Format: engine.AgentOutputSchema(),
	}, &out, engine.Hooks{})
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if res.Partial {
		t.Fatal("Layer 1 success must not be marked partial")
	}
	if out.AgentID != "engineering-software-architect" {
		t.Fatalf("agent id = %q", out.AgentID)
	}
	if out.Status != "done" {
		t.Fatalf("status = %q", out.Status)
	}
	if out.Output.Summary == "" || len(out.Output.Findings) != 3 {
		t.Fatalf("output not parsed: %+v", out.Output)
	}
	if out.Output.Findings[0].Severity != models.SeverityCritical {
		t.Fatalf("finding severity = %q", out.Output.Findings[0].Severity)
	}
	if out.Thinking != "" {
		t.Fatalf("no think block sent; thinking = %q", out.Thinking)
	}
	if n := mock.CallCount(func(c RecordedCall) bool { return c.HasFormat }); n != 1 {
		t.Fatalf("format-constrained calls = %d, want 1", n)
	}
}

// TestJSONThinkingBlock verifies extraction of <think>...</think> content and
// stripping before parse.
func TestJSONThinkingBlock(t *testing.T) {
	mock := NewMockLLM()
	defer mock.Server.Close()

	fixture := loadFixture(t, "agent_output.json")
	content := ThinkOpen + "\nI should analyze sync risks before answering.\n" + ThinkClose + "\n" + fixture
	mock.Register(Always, func(MockRequest) string { return content })

	client := engine.NewClient()
	var out models.AgentOutput
	res, err := client.JSON(context.Background(), runtimeFor(mock), engine.ChatRequest{
		Format: engine.AgentOutputSchema(),
	}, &out, engine.Hooks{})
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if res.Thinking != "I should analyze sync risks before answering." {
		t.Fatalf("thinking = %q", res.Thinking)
	}
	if out.AgentID != "engineering-software-architect" {
		t.Fatalf("thinking was not stripped before parse; agent id = %q", out.AgentID)
	}
	// The envelope's thinking field is a placeholder the model never writes
	// into — the runner copies res.Thinking into the persisted output.
	if out.Thinking != "" {
		t.Fatalf("model placeholder thinking should stay empty, got %q", out.Thinking)
	}
}

// TestJSONRetryWithCorrection covers Layer 2: malformed JSON triggers the
// correction retry loop (max 2) before falling through.
func TestJSONRetryWithCorrection(t *testing.T) {
	mock := NewMockLLM()
	defer mock.Server.Close()

	fixture := loadFixture(t, "agent_output.json")
	attempts := 0
	mock.Register(Always, func(MockRequest) string {
		attempts++
		if attempts <= 2 {
			return "this is not json at all"
		}
		return fixture
	})

	client := engine.NewClient()
	var out models.AgentOutput
	if _, err := client.JSON(context.Background(), runtimeFor(mock), engine.ChatRequest{
		Format: engine.AgentOutputSchema(),
	}, &out, engine.Hooks{}); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if got := mock.CallCount(func(RecordedCall) bool { return true }); got != 3 {
		t.Fatalf("provider calls = %d, want 3 (original + 2 corrections)", got)
	}
	calls := mock.Calls()
	last := calls[len(calls)-1]
	if !strings.Contains(last.Last, "not valid JSON") {
		t.Fatalf("correction prompt missing from last user message: %q", last.Last)
	}
	if len(last.Messages) < 2 || last.Messages[len(last.Messages)-2].Role != "assistant" {
		t.Fatalf("bad assistant response not appended to conversation")
	}
	if out.AgentID != "engineering-software-architect" {
		t.Fatalf("agent id = %q", out.AgentID)
	}
}

// TestJSONMarkdownExtractionPartial covers Layer 3: JSON buried in prose is
// extracted, parsed, and marked partial — never dropped.
func TestJSONMarkdownExtractionPartial(t *testing.T) {
	mock := NewMockLLM()
	defer mock.Server.Close()

	fixture := loadFixture(t, "agent_output.json")
	mock.Register(Always, func(MockRequest) string {
		return "Sure! Here is the JSON you asked for:\n\n" + fixture + "\n\nHope that helps!"
	})

	client := engine.NewClient()
	var out models.AgentOutput
	res, err := client.JSON(context.Background(), runtimeFor(mock), engine.ChatRequest{
		Format: engine.AgentOutputSchema(),
	}, &out, engine.Hooks{})
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if !res.Partial {
		t.Fatal("Layer 3 success must be marked partial")
	}
	if out.AgentID != "engineering-software-architect" {
		t.Fatalf("agent id = %q", out.AgentID)
	}
}

// TestFormatRejectedDegradation: when the provider rejects the format
// parameter, the call degrades to Layer 2 rather than failing.
func TestFormatRejectedDegradation(t *testing.T) {
	mock := NewMockLLM()
	defer mock.Server.Close()

	fixture := loadFixture(t, "agent_output.json")
	mock.Register(Always, func(MockRequest) string { return fixture })
	mock.SetRejectFormat(true)

	client := engine.NewClient()
	var out models.AgentOutput
	if _, err := client.JSON(context.Background(), runtimeFor(mock), engine.ChatRequest{
		Format: engine.AgentOutputSchema(),
	}, &out, engine.Hooks{}); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	calls := mock.Calls()
	if len(calls) != 2 {
		t.Fatalf("calls = %d, want 2 (rejected + unconstrained retry)", len(calls))
	}
	if !calls[0].HasFormat {
		t.Fatal("first call should have carried the format schema")
	}
	if calls[1].HasFormat {
		t.Fatal("second call should have dropped the format schema")
	}
	if out.AgentID != "engineering-software-architect" {
		t.Fatalf("agent id = %q", out.AgentID)
	}
}

// TestUnreachable verifies transport failures surface as UnreachableError
// (mapped to 502 by handlers).
func TestUnreachable(t *testing.T) {
	client := engine.NewClient()
	cfg := engine.RuntimeLLMConfig{Endpoint: "http://127.0.0.1:1", Model: "m"}
	_, err := client.Complete(context.Background(), cfg, engine.ChatRequest{}, engine.Hooks{})
	var ue *engine.UnreachableError
	if !errors.As(err, &ue) {
		t.Fatalf("want UnreachableError, got %v", err)
	}
}

// TestStreamingHooks verifies streamed deltas separate thinking from content.
func TestStreamingHooks(t *testing.T) {
	mock := NewMockLLM()
	defer mock.Server.Close()

	content := ThinkOpen + "internal reasoning" + ThinkClose + "final answer text"
	mock.Register(Always, func(MockRequest) string { return content })

	var thinkSb, contentSb strings.Builder
	client := engine.NewClient()
	res, err := client.Complete(context.Background(), runtimeFor(mock), engine.ChatRequest{
		Stream: true,
	}, engine.Hooks{
		OnThinking: func(c string) { thinkSb.WriteString(c) },
		OnChunk:    func(c string) { contentSb.WriteString(c) },
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if res.Thinking != "internal reasoning" {
		t.Fatalf("thinking = %q", res.Thinking)
	}
	if res.Content != "final answer text" {
		t.Fatalf("content = %q", res.Content)
	}
	if thinkSb.String() != "internal reasoning" {
		t.Fatalf("streamed thinking = %q", thinkSb.String())
	}
	if contentSb.String() != "final answer text" {
		t.Fatalf("streamed content = %q", contentSb.String())
	}
}

// TestSplitThinkingCases covers the edge cases of think-block extraction.
func TestSplitThinkingCases(t *testing.T) {
	cases := []struct {
		name        string
		in          string
		wantContent string
		wantThink   string
	}{
		{"no block", "plain answer", "plain answer", ""},
		{"leading block", ThinkOpen + "t" + ThinkClose + "answer", "answer", "t"},
		{"unterminated", ThinkOpen + "all reasoning", "", "all reasoning"},
		{"trailing block", "answer " + ThinkOpen + "t" + ThinkClose, "answer", "t"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := engine.SplitThinking(tc.in)
			if err != nil {
				t.Fatalf("SplitThinking: %v", err)
			}
			if res.Content != tc.wantContent {
				t.Fatalf("content = %q, want %q", res.Content, tc.wantContent)
			}
			if res.Thinking != tc.wantThink {
				t.Fatalf("thinking = %q, want %q", res.Thinking, tc.wantThink)
			}
		})
	}
}

// TestExtractJSONObject covers the Layer 3 scanner.
func TestExtractJSONObject(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{"plain", `{"a":1}`, `{"a":1}`, true},
		{"embedded", `Sure: {"a":{"b":"}"}}, trailing`, `{"a":{"b":"}"}}`, true},
		{"escaped quote", `{"a":"x\"}"}`, `{"a":"x\"}"}`, true},
		{"none", "no json here", "", false},
		{"unclosed", `{"a":1`, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := engine.ExtractJSONObject(tc.in)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("ExtractJSONObject(%q) = %q,%v; want %q,%v", tc.in, got, ok, tc.want, tc.ok)
			}
		})
	}
}
