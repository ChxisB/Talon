package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// ── MCP Types ───────────────────────────────────────

type MCPServerConfig struct {
	Type        string            `json:"type"` // "local" or "remote"
	Command     []string          `json:"command,omitempty"`
	Cwd         string            `json:"cwd,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	URL         string            `json:"url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Disabled    bool              `json:"disabled,omitempty"`
	Timeout     int               `json:"timeout,omitempty"`
}

type MCPConfig struct {
	Servers map[string]MCPServerConfig `json:"servers,omitempty"`
}

type MCPTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema any         `json:"inputSchema"`
	ServerID    string      `json:"serverID"`
}

type MCPCallRequest struct {
	ServerID string `json:"serverID"`
	ToolName string `json:"toolName"`
	Args     any    `json:"args"`
}

// ── JSON-RPC Types ──────────────────────────────────

type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ── MCP Server Instance ─────────────────────────────

type MCPServerInstance struct {
	Config   MCPServerConfig
	Name     string
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   *bufio.Scanner
	mu       sync.Mutex
	seq      int
	tools    []MCPTool
	running  bool
	stopCh   chan struct{}
}

type MCPServerStore struct {
	mu      sync.Mutex
	servers map[string]*MCPServerInstance
}

var GlobalMCPServers = &MCPServerStore{
	servers: make(map[string]*MCPServerInstance),
}

// ── Load config from talon.json ─────────────────────

func LoadMCPConfig(path string) (*MCPConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config struct {
		MCP *MCPConfig `json:"mcp"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if config.MCP == nil {
		return &MCPConfig{}, nil
	}
	return config.MCP, nil
}

func FindMCPConfig(startDir string) *MCPConfig {
	// Search for talon.json or opencode.json in parent directories
	dir := startDir
	for i := 0; i < 10; i++ {
		for _, name := range []string{"talon.json", "talon.jsonc", "opencode.json", "opencode.jsonc"} {
			path := filepath.Join(dir, name)
			config, err := LoadMCPConfig(path)
			if err == nil && len(config.Servers) > 0 {
				return config
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return &MCPConfig{}
}

// ── Manage MCP Servers ──────────────────────────────

func (s *MCPServerStore) StartAll(config *MCPConfig, cwd string) []error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []error
	for name, cfg := range config.Servers {
		if cfg.Disabled {
			continue
		}

		inst := &MCPServerInstance{
			Config:  cfg,
			Name:    name,
			stopCh:  make(chan struct{}),
		}

		switch cfg.Type {
		case "local":
			if err := inst.startLocal(cwd); err != nil {
				errs = append(errs, fmt.Errorf("mcp %s: %w", name, err))
				continue
			}
		case "remote":
			inst.running = true
		}

		// Discover tools
		if inst.running {
			if err := inst.discoverTools(); err != nil {
				errs = append(errs, fmt.Errorf("mcp %s tool discovery: %w", name, err))
			}
		}

		s.servers[name] = inst
	}

	return errs
}

func (s *MCPServerStore) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for name, inst := range s.servers {
		inst.stop()
		delete(s.servers, name)
	}
}

func (s *MCPServerStore) GetAllTools() []MCPTool {
	s.mu.Lock()
	defer s.mu.Unlock()

	var tools []MCPTool
	for _, inst := range s.servers {
		if !inst.running {
			continue
		}
		tools = append(tools, inst.tools...)
	}
	return tools
}

func (s *MCPServerStore) CallTool(req MCPCallRequest) (json.RawMessage, error) {
	s.mu.Lock()
	inst, ok := s.servers[req.ServerID]
	s.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("MCP server %s not found", req.ServerID)
	}

	switch inst.Config.Type {
	case "local":
		return inst.callLocal(req)
	case "remote":
		return inst.callRemote(req)
	default:
		return nil, fmt.Errorf("unknown MCP server type: %s", inst.Config.Type)
	}
}

// ── Local Server ────────────────────────────────────

func (inst *MCPServerInstance) startLocal(cwd string) error {
	if len(inst.Config.Command) == 0 {
		return fmt.Errorf("no command configured")
	}

	cmd := exec.Command(inst.Config.Command[0], inst.Config.Command[1:]...)
	if inst.Config.Cwd != "" {
		cmd.Dir = inst.Config.Cwd
	} else {
		cmd.Dir = cwd
	}

	cmd.Env = os.Environ()
	for k, v := range inst.Config.Environment {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	inst.cmd = cmd
	inst.stdin = stdin
	inst.stdout = bufio.NewScanner(stdout)
	inst.stdout.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	inst.running = true

	// Monitor for exit
	go func() {
		cmd.Wait()
		inst.mu.Lock()
		inst.running = false
		inst.mu.Unlock()
	}()

	return nil
}

func (inst *MCPServerInstance) discoverTools() error {
	resp, err := inst.sendRequest("tools/list", nil)
	if err != nil {
		return fmt.Errorf("list tools: %w", err)
	}

	var result struct {
		Tools []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			InputSchema any    `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("parse tools: %w", err)
	}

	inst.tools = nil
	for _, t := range result.Tools {
		inst.tools = append(inst.tools, MCPTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
			ServerID:    inst.Name,
		})
	}

	return nil
}

func (inst *MCPServerInstance) sendRequest(method string, params any) (json.RawMessage, error) {
	inst.mu.Lock()
	inst.seq++
	id := inst.seq
	inst.mu.Unlock()

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if _, err := inst.stdin.Write(append(data, '\n')); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	// Read response
	timeout := 30 * time.Second
	if inst.Config.Timeout > 0 {
		timeout = time.Duration(inst.Config.Timeout) * time.Second
	}

	done := make(chan json.RawMessage, 1)
	errCh := make(chan error, 1)

	go func() {
		for inst.stdout.Scan() {
			line := inst.stdout.Text()

			// Server may send notifications (no ID) — skip
			var resp JSONRPCResponse
			if err := json.Unmarshal([]byte(line), &resp); err != nil {
				continue
			}

			if resp.ID == id {
				if resp.Error != nil {
					errCh <- fmt.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
					return
				}
				done <- resp.Result
				return
			}
		}
		errCh <- fmt.Errorf("MCP connection closed")
	}()

	select {
	case result := <-done:
		return result, nil
	case err := <-errCh:
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("MCP request timed out after %v", timeout)
	}
}

func (inst *MCPServerInstance) callLocal(req MCPCallRequest) (json.RawMessage, error) {
	return inst.sendRequest("tools/call", map[string]any{
		"name":      req.ToolName,
		"arguments": req.Args,
	})
}

func (inst *MCPServerInstance) callRemote(req MCPCallRequest) (json.RawMessage, error) {
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      req.ToolName,
			"arguments": req.Args,
		},
	})

	httpReq, err := http.NewRequest("POST", inst.Config.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range inst.Config.Headers {
		httpReq.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rpcResp JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("MCP error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

func (inst *MCPServerInstance) stop() {
	inst.mu.Lock()
	defer inst.mu.Unlock()

	if !inst.running {
		return
	}
	inst.running = false

	if inst.stdin != nil {
		inst.stdin.Close()
	}
	if inst.cmd != nil {
		inst.cmd.Process.Kill()
	}
	close(inst.stopCh)
}
