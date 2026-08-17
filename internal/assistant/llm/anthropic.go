package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// AnthropicConfig carries everything needed to reach the Messages API. Values
// come from application configuration; nothing here is hardcoded so an operator
// can point at a proxy or change models without a rebuild.
type AnthropicConfig struct {
	APIKey    string
	Model     string
	BaseURL   string // default https://api.anthropic.com
	MaxTokens int    // per-response output budget
	// Timeout bounds one HTTP attempt. The orchestrator's context bounds the
	// whole conversation turn including retries.
	Timeout    time.Duration
	MaxRetries int
}

// Anthropic implements Client over the Messages API using only the standard
// library. The official SDK is deliberately not used: this keeps the request
// surface auditable and adds no new dependency to the supply chain.
type Anthropic struct {
	cfg  AnthropicConfig
	http *http.Client
}

const anthropicVersion = "2023-06-01"

func NewAnthropic(cfg AnthropicConfig) *Anthropic {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.anthropic.com"
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 2048
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	return &Anthropic{
		cfg: cfg,
		// The per-attempt timeout lives on the request context, not the
		// http.Client, so a caller-supplied deadline shorter than cfg.Timeout
		// still wins.
		http: &http.Client{},
	}
}

// wire structures — only what this application uses, nothing speculative.

type anthropicRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
	Tools     []Tool    `json:"tools,omitempty"`
}

type anthropicResponse struct {
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      Usage          `json:"usage"`
}

type anthropicError struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Complete performs one Messages call with bounded retries on transient
// failures. Retrying here is safe by construction: a completion has no side
// effects — every state change in the assistant happens in our own tool and
// approval layers, never inside the provider call.
func (a *Anthropic) Complete(ctx context.Context, req Request) (*Response, error) {
	if a.cfg.APIKey == "" {
		return nil, fmt.Errorf("%w: no API key configured", ErrUnavailable)
	}

	body, err := json.Marshal(anthropicRequest{
		Model:     a.cfg.Model,
		MaxTokens: a.cfg.MaxTokens,
		System:    req.System,
		Messages:  req.Messages,
		Tools:     req.Tools,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: marshal: %v", ErrBadRequest, err)
	}

	var lastErr error
	for attempt := 0; attempt <= a.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			// Full jitter keeps concurrent callers from retrying in lockstep.
			backoff := time.Duration(rand.Int63n(int64(time.Second) << uint(attempt-1)))
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("%w: %v", ErrUnavailable, ctx.Err())
			case <-time.After(backoff):
			}
		}

		resp, retryable, err := a.attempt(ctx, body)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !retryable {
			return nil, err
		}
	}
	return nil, lastErr
}

// attempt runs a single HTTP call. The second return value reports whether the
// failure is worth retrying.
func (a *Anthropic) attempt(ctx context.Context, body []byte) (*Response, bool, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, a.cfg.Timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(attemptCtx, http.MethodPost,
		strings.TrimRight(a.cfg.BaseURL, "/")+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-Key", a.cfg.APIKey)
	httpReq.Header.Set("Anthropic-Version", anthropicVersion)

	httpResp, err := a.http.Do(httpReq)
	if err != nil {
		// Timeouts and connection failures are transient unless the parent
		// context is done, in which case retrying would just overrun it.
		if ctx.Err() != nil {
			return nil, false, fmt.Errorf("%w: %v", ErrUnavailable, ctx.Err())
		}
		return nil, true, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	// The cap protects against a misbehaving upstream; real responses are far
	// smaller because MaxTokens bounds them.
	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, 4<<20))
	if err != nil {
		return nil, true, fmt.Errorf("%w: read response: %v", ErrUnavailable, err)
	}

	if httpResp.StatusCode != http.StatusOK {
		retryable, cErr := a.classify(httpResp.StatusCode, respBody)
		return nil, retryable, cErr
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, true, fmt.Errorf("%w: decode response: %v", ErrUnavailable, err)
	}
	return &Response{Content: parsed.Content, StopReason: parsed.StopReason, Usage: parsed.Usage}, false, nil
}

// classify maps a non-200 status to the package error taxonomy. Error bodies
// are summarised, never logged verbatim at call sites, and API keys never
// appear in them.
func (a *Anthropic) classify(status int, body []byte) (bool, error) {
	var apiErr anthropicError
	_ = json.Unmarshal(body, &apiErr)
	detail := apiErr.Error.Message
	if detail == "" {
		detail = http.StatusText(status)
	}
	// The provider's message can mention model names and limits but never our
	// request content; still, keep what we propagate short.
	if len(detail) > 300 {
		detail = detail[:300]
	}

	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return false, fmt.Errorf("%w: %s", ErrAuth, detail)
	case status == http.StatusTooManyRequests || status >= 500:
		// 529 (overloaded) also lands here.
		return true, fmt.Errorf("%w: HTTP %d: %s", ErrUnavailable, status, detail)
	case status >= 400 && status < 500:
		// Our request was malformed. Log once at the source with the type tag
		// so a schema bug is visible in operations without dumping payloads.
		log.Printf("assistant: model request rejected (HTTP %d, type=%s): %s", status, apiErr.Error.Type, detail)
		return false, fmt.Errorf("%w: HTTP %d: %s", ErrBadRequest, status, detail)
	default:
		return true, fmt.Errorf("%w: HTTP %d: %s", ErrUnavailable, status, detail)
	}
}

// Model reports the configured model identifier for observability records.
func (a *Anthropic) Model() string { return a.cfg.Model }

var _ Client = (*Anthropic)(nil)

// Unavailable reports whether err is a transient provider failure, so handlers
// can choose a 503 and the UI can offer a retry.
func Unavailable(err error) bool { return errors.Is(err, ErrUnavailable) }
