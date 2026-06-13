// Package messaging provides multi-platform messaging adapters for the
// Spectre Proxy agent. Ported from free-claude-code/messaging/.
package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ChxisB/spectre-proxy/agent/internal/config"
	"github.com/ChxisB/spectre-proxy/agent/internal/protocol"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
)

// Platform defines the interface for messaging platform adapters.
type Platform interface {
	// Name returns the platform name (e.g. "discord", "telegram").
	Name() string
	// Start begins listening for messages. Blocks until context is cancelled.
	Start(ctx context.Context) error
	// Send sends a message to a conversation.
	Send(conversationID string, text string) error
}

// Message represents an incoming message from any platform.
type Message struct {
	Platform       string
	ConversationID string
	UserID         string
	Text           string
	IsReply        bool
	ReplyToID      string
}

// Session tracks a conversation across turns.
type Session struct {
	ID             string
	Platform       string
	ConversationID string
	Messages       []protocol.Message
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Handler processes incoming messages from all platforms.
type Handler struct {
	mu        sync.RWMutex
	settings  *config.Settings
	provider  providers.Provider
	registry  *providers.Registry
	sessions  map[string]*Session
	platforms []Platform
}

// NewHandler creates a new message handler.
func NewHandler(cfg *config.Settings, reg *providers.Registry) *Handler {
	return &Handler{
		settings: cfg,
		registry: reg,
		sessions: make(map[string]*Session),
	}
}

// AddPlatform registers a messaging platform.
func (h *Handler) AddPlatform(p Platform) {
	h.platforms = append(h.platforms, p)
}

// Start starts all registered platforms.
func (h *Handler) Start(ctx context.Context) error {
	for _, p := range h.platforms {
		go func(platform Platform) {
			log.Printf("Starting messaging platform: %s", platform.Name())
			if err := platform.Start(ctx); err != nil {
				log.Printf("Platform %s error: %v", platform.Name(), err)
			}
		}(p)
	}
	return nil
}

// HandleMessage processes an incoming message from any platform.
func (h *Handler) HandleMessage(ctx context.Context, msg Message) error {
	sessionID := fmt.Sprintf("%s:%s", msg.Platform, msg.ConversationID)

	h.mu.Lock()
	session, exists := h.sessions[sessionID]
	if !exists {
		session = &Session{
			ID:             sessionID,
			Platform:       msg.Platform,
			ConversationID: msg.ConversationID,
			CreatedAt:      time.Now(),
		}
		h.sessions[sessionID] = session
	}
	session.UpdatedAt = time.Now()
	h.mu.Unlock()

	// Add user message
	userContent, _ := json.Marshal([]protocol.ContentBlock{
		{Type: "text", Text: msg.Text},
	})
	session.Messages = append(session.Messages, protocol.Message{
		Role:    "user",
		Content: userContent,
	})

	// Get a provider for the response
	modelRef := h.settings.ResolveModel("")
	providerID := config.ParseProviderType(modelRef)
	providerModel := config.ParseModelName(modelRef)

	provider, err := h.registry.Get(providerID, providers.DefaultProviderConfig(), h.settings)
	if err != nil {
		return fmt.Errorf("no provider available: %w", err)
	}

	// Make the AI request
	req := &protocol.MessagesRequest{
		Model:    providerModel,
		Messages: session.Messages,
	}

	events, err := provider.StreamResponse(ctx, req, 0, true)
	if err != nil {
		return fmt.Errorf("ai request failed: %w", err)
	}

	// Accumulate response text
	var responseText string
	for evt := range events {
		if evt.Type == "content_block_delta" {
			var delta protocol.ContentBlockDelta
			if err := json.Unmarshal(evt.Data, &delta); err == nil {
				responseText += delta.Delta.Text
			}
		}
	}

	// Add assistant response to session
	assistantContent, _ := json.Marshal([]protocol.ContentBlock{
		{Type: "text", Text: responseText},
	})
	session.Messages = append(session.Messages, protocol.Message{
		Role:    "assistant",
		Content: assistantContent,
	})

	// Send response back through the platform
	for _, p := range h.platforms {
		if p.Name() == msg.Platform {
			return p.Send(msg.ConversationID, responseText)
		}
	}

	return nil
}
