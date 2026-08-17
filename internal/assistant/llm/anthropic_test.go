package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func testClient(t *testing.T, handler http.HandlerFunc, retries int) *Anthropic {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewAnthropic(AnthropicConfig{
		APIKey:     "test-key",
		Model:      "test-model",
		BaseURL:    srv.URL,
		MaxTokens:  128,
		Timeout:    2 * time.Second,
		MaxRetries: retries,
	})
}

func okBody(t *testing.T, w http.ResponseWriter, resp anthropicResponse) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Errorf("encode: %v", err)
	}
}

func TestCompleteParsesTextAndUsage(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-key" {
			t.Errorf("missing api key header")
		}
		if r.Header.Get("Anthropic-Version") == "" {
			t.Errorf("missing version header")
		}
		var req anthropicRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Model != "test-model" || len(req.Messages) != 1 {
			t.Errorf("unexpected request: %+v", req)
		}
		okBody(t, w, anthropicResponse{
			Content:    []ContentBlock{{Type: BlockText, Text: "مرحبا"}},
			StopReason: StopEndTurn,
			Usage:      Usage{InputTokens: 10, OutputTokens: 5},
		})
	}, 0)

	resp, err := c.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("hi")}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StopReason != StopEndTurn || resp.Content[0].Text != "مرحبا" {
		t.Fatalf("resp = %+v", resp)
	}
	if resp.Usage.InputTokens != 10 {
		t.Fatalf("usage = %+v", resp.Usage)
	}
}

func TestCompleteParsesToolUse(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		okBody(t, w, anthropicResponse{
			Content: []ContentBlock{
				{Type: BlockText, Text: "checking"},
				{Type: BlockToolUse, ID: "tu_1", Name: "get_current_shift", Input: json.RawMessage(`{}`)},
			},
			StopReason: StopToolUse,
		})
	}, 0)

	resp, err := c.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("shift?")}}},
		Tools:    []Tool{{Name: "get_current_shift", InputSchema: json.RawMessage(`{"type":"object"}`)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StopReason != StopToolUse {
		t.Fatalf("stop = %s", resp.StopReason)
	}
	if resp.Content[1].Name != "get_current_shift" || resp.Content[1].ID != "tu_1" {
		t.Fatalf("tool block = %+v", resp.Content[1])
	}
}

func TestRetryOn429ThenSuccess(t *testing.T) {
	var calls int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"type":"rate_limit_error","message":"slow down"}}`))
			return
		}
		okBody(t, w, anthropicResponse{Content: []ContentBlock{TextBlock("ok")}, StopReason: StopEndTurn})
	}, 2)

	resp, err := c.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("x")}}}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content[0].Text != "ok" || atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("calls=%d resp=%+v", calls, resp)
	}
}

func TestAuthFailureDoesNotRetry(t *testing.T) {
	var calls int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"type":"authentication_error","message":"bad key"}}`))
	}, 3)

	_, err := c.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("x")}}}})
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("err = %v", err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("auth failure retried %d times", calls)
	}
}

func TestBadRequestDoesNotRetry(t *testing.T) {
	var calls int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"type":"invalid_request_error","message":"bad schema"}}`))
	}, 3)

	_, err := c.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("x")}}}})
	if !errors.Is(err, ErrBadRequest) {
		t.Fatalf("err = %v", err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("bad request retried %d times", calls)
	}
}

func TestOverloadedRetriesThenUnavailable(t *testing.T) {
	var calls int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(529)
		_, _ = w.Write([]byte(`{"error":{"type":"overloaded_error","message":"overloaded"}}`))
	}, 2)

	_, err := c.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("x")}}}})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v", err)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("expected 3 attempts, got %d", calls)
	}
}

func TestContextCancellationStopsRetries(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}, 5)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := c.Complete(ctx, Request{Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("x")}}}})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("retries outlived the context")
	}
}

func TestMissingAPIKeyFailsFast(t *testing.T) {
	c := NewAnthropic(AnthropicConfig{})
	_, err := c.Complete(context.Background(), Request{})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v", err)
	}
}
