package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/assistant"
	"shiftmaster-backend/internal/assistant/llm"
)

// AssistantHandler exposes the AI assistant: a streaming chat endpoint and the
// approval endpoints that form the human side of the action security boundary.
type AssistantHandler struct {
	svc *assistant.Service
}

func NewAssistantHandler(svc *assistant.Service) *AssistantHandler {
	return &AssistantHandler{svc: svc}
}

// Status lets the frontend decide whether to render the assistant at all.
// Nothing sensitive: enabled yes/no only.
func (h *AssistantHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"enabled": h.svc.Enabled()}})
}

type chatRequest struct {
	ConversationID *uuid.UUID `json:"conversation_id"`
	Message        string     `json:"message" binding:"required"`
}

// Chat runs one assistant turn and streams events as SSE. Validation and rate
// limiting happen before the stream opens, so those failures are ordinary
// HTTP statuses; anything after the first byte arrives as an "error" event.
func (h *AssistantHandler) Chat(c *gin.Context) {
	actor, ok := actorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "authentication required"})
		return
	}

	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	release, err := h.svc.Begin(actor, req.Message)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, assistant.ErrDisabled):
			status = http.StatusServiceUnavailable
		case errors.Is(err, assistant.ErrRateLimited):
			status = http.StatusTooManyRequests
		case errors.Is(err, assistant.ErrBusy):
			status = http.StatusConflict
		case errors.Is(err, assistant.ErrInputTooLong):
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"success": false, "error": err.Error()})
		return
	}
	defer release()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	flusher, canFlush := c.Writer.(http.Flusher)

	emit := func(ev assistant.Event) {
		payload, err := json.Marshal(ev)
		if err != nil {
			return
		}
		c.SSEvent("message", string(payload))
		if canFlush {
			flusher.Flush()
		}
	}

	result, err := h.svc.HandleMessage(c.Request.Context(), actor, req.ConversationID, req.Message, emit)
	if err != nil {
		emit(assistant.Event{Type: "error", Message: chatErrorMessage(err)})
	}
	done := assistant.Event{Type: "done"}
	if result != nil {
		done.ConversationID = result.ConversationID.String()
	}
	emit(done)
}

// chatErrorMessage maps turn failures to something safe and useful for the
// UI. Internal detail stays in the server log.
func chatErrorMessage(err error) string {
	switch {
	case errors.Is(err, llm.ErrUnavailable):
		return "assistant_unavailable"
	case errors.Is(err, llm.ErrAuth), errors.Is(err, llm.ErrBadRequest):
		return "assistant_misconfigured"
	case errors.Is(err, assistant.ErrConversation):
		return "conversation_unavailable"
	default:
		return "assistant_error"
	}
}

// Decide handles both approval and rejection of one pending action.
func (h *AssistantHandler) decide(c *gin.Context, approve bool) {
	actor, ok := actorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "authentication required"})
		return
	}
	actionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid action id"})
		return
	}

	outcome, err := h.svc.Decide(c.Request.Context(), actor, actionID, approve)
	if err != nil {
		if errors.Is(err, assistant.ErrActionNotDecidable) {
			body := gin.H{"success": false, "error": "action is no longer pending"}
			if outcome != nil && outcome.Action != nil {
				body["data"] = gin.H{"action": outcome.Action}
			}
			c.JSON(http.StatusConflict, body)
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "not authorized"})
		return
	}

	data := gin.H{"action": outcome.Action}
	if outcome.Result != nil {
		data["result"] = outcome.Result
	}
	if outcome.FailureReason != "" {
		data["failure_reason"] = outcome.FailureReason
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *AssistantHandler) Approve(c *gin.Context) { h.decide(c, true) }
func (h *AssistantHandler) Reject(c *gin.Context)  { h.decide(c, false) }

// GetAction returns one pending action's current state (card reload).
func (h *AssistantHandler) GetAction(c *gin.Context) {
	actor, ok := actorID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "authentication required"})
		return
	}
	actionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid action id"})
		return
	}
	action, err := h.svc.PendingAction(c.Request.Context(), actor, actionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "action not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"action": action}})
}
