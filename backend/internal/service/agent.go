package service

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/talon/backend/internal/db"
	"github.com/talon/backend/internal/llm"
)

const (
	MaxToolIterations = 25
	SystemPrompt = `You are Talon, an AI-powered coding assistant. You help users with software development tasks.

You have access to tools that let you:
- Read and write files
- Run bash commands
- Search through code

Rules:
1. Think step by step about what the user needs
2. Use tools to accomplish tasks
3. Explain your actions clearly
4. If you're unsure, ask the user for clarification
5. Always check file contents before making changes
6. Run commands to verify your work`

	DefaultModel = "gpt-4o"
)

// ── Agent Events ────────────────────────────────────

type AgentEvent struct {
	Type    AgentEventType `json:"type"`
	Content string         `json:"content,omitempty"`
	Tool    *ToolExecEvent `json:"tool,omitempty"`
	Done    bool           `json:"done,omitempty"`
	Error   string         `json:"error,omitempty"`
}

type AgentEventType string

const (
	AgentEventText  AgentEventType = "text"
	AgentEventTool  AgentEventType = "tool"
	AgentEventDone  AgentEventType = "done"
	AgentEventError AgentEventType = "error"
)

type ToolExecEvent struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Result    string `json:"result"`
	Error     string `json:"error,omitempty"`
}

// ── Agent Session ───────────────────────────────────

type AgentSession struct {
	ID         string
	Model      string
	Messages   []llm.Message
	ToolRegistry *ToolRegistry
	mu         sync.Mutex
	running    bool
	events     chan AgentEvent
}

type AgentSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*AgentSession
}

var GlobalAgents = &AgentSessionStore{
	sessions: make(map[string]*AgentSession),
}

func (s *AgentSessionStore) GetOrCreate(id, model string) *AgentSession {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.sessions[id]; ok {
		return existing
	}

	toolRegistry := NewToolRegistry()

	// Load MCP tools from config
	mcpConfig := FindMCPConfig(".")
	if mcpConfig != nil {
		GlobalMCPServers.StartAll(mcpConfig, ".")
		for _, t := range GlobalMCPServers.GetAllTools() {
			toolName := t.Name
			toolDesc := t.Description
			toolSchema := t.InputSchema
			serverID := t.ServerID

			toolRegistry.Register(ToolDef{
				Name:        "mcp_" + toolName,
				Description: toolDesc + " (MCP: " + serverID + ")",
				Parameters:  toolSchema,
				Execute: func(args json.RawMessage) (string, error) {
					result, err := GlobalMCPServers.CallTool(MCPCallRequest{
						ServerID: serverID,
						ToolName: toolName,
						Args:     args,
					})
					if err != nil {
						return "", err
					}
					return string(result), nil
				},
			})
		}
	}

	agent := &AgentSession{
		ID:           id,
		Model:        model,
		Messages:     []llm.Message{},
		ToolRegistry: toolRegistry,
		events:       make(chan AgentEvent, 256),
	}
	s.sessions[id] = agent
	return agent
}

func (s *AgentSessionStore) Get(id string) *AgentSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[id]
}

func (s *AgentSessionStore) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

// ── Agent Execution ─────────────────────────────────

func (a *AgentSession) Run(prompt string) <-chan AgentEvent {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return nil
	}
	a.running = true
	a.mu.Unlock()

	go a.executeLoop(prompt)
	return a.events
}

func (a *AgentSession) executeLoop(prompt string) {
	defer close(a.events)
	defer func() {
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
	}()

	// Add user message
	a.Messages = append(a.Messages, llm.Message{
		Role:    llm.RoleUser,
		Content: prompt,
	})

	// Save input as session_message
	a.saveMessage("user", prompt)

	model := a.Model
	if model == "" {
		model = DefaultModel
	}

	provider, _, err := llm.GetProvider(model)
	if err != nil {
		a.events <- AgentEvent{Type: AgentEventError, Error: err.Error()}
		return
	}

	for iteration := 0; iteration < MaxToolIterations; iteration++ {
		req := llm.CompletionRequest{
			Model:    model,
			Messages: append([]llm.Message{{Role: llm.RoleSystem, Content: SystemPrompt}}, a.Messages...),
			Tools:    a.ToolRegistry.Definitions(),
			MaxTokens: 4096,
		}

		resp, err := provider.Complete(req)
		if err != nil {
			a.events <- AgentEvent{Type: AgentEventError, Error: err.Error()}
			return
		}

		// Add assistant response to messages
		assistantMsg := llm.Message{
			Role:    llm.RoleAssistant,
			Content: resp.Content,
		}

		if len(resp.ToolCalls) > 0 {
			assistantMsg.ToolCalls = resp.ToolCalls
		}

		a.Messages = append(a.Messages, assistantMsg)

		// Stream text content
		if resp.Content != "" {
			a.events <- AgentEvent{Type: AgentEventText, Content: resp.Content}
			a.saveMessage("assistant", resp.Content)
		}

		// Check if the LLM wants to use tools
		if len(resp.ToolCalls) == 0 {
			a.events <- AgentEvent{Type: AgentEventDone, Done: true}
			return
		}

		// Execute tools
		for _, tc := range resp.ToolCalls {
			a.events <- AgentEvent{
				Type: AgentEventTool,
				Tool: &ToolExecEvent{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}

			result, err := a.ToolRegistry.Execute(tc.Function.Name, tc.Function.Arguments)
			toolResult := ToolExecEvent{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}

			if err != nil {
				toolResult.Error = err.Error()
				a.events <- AgentEvent{Type: AgentEventTool, Tool: &toolResult}
			} else {
				toolResult.Result = result
				a.events <- AgentEvent{Type: AgentEventTool, Tool: &toolResult}
			}

			// Add tool result to messages
			a.Messages = append(a.Messages, llm.Message{
				Role:       llm.RoleTool,
				ToolCallID: tc.ID,
				Content:    result,
			})
		}
	}

	a.events <- AgentEvent{Type: AgentEventDone, Done: true}
}

func (a *AgentSession) saveMessage(role, content string) {
	now := time.Now().UnixMilli()
	var seq int
	db.DB.QueryRow("SELECT COALESCE(MAX(seq), 0) + 1 FROM session_message WHERE session_id = ?", a.ID).Scan(&seq)

	id := fmt.Sprintf("msg_%s_%d", a.ID[:8], seq)
	data := map[string]any{
		"role":    role,
		"content": content,
	}
	dataJSON, _ := json.Marshal(data)

	db.DB.Exec(
		"INSERT INTO session_message (id, session_id, type, seq, time_created, time_updated, data) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, a.ID, role, seq, now, now, string(dataJSON),
	)
}
