package service

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/google/uuid"
)

// ── PTY Session ─────────────────────────────────────

type PTYInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Directory   string `json:"directory"`
	Command     string `json:"command"`
	Pid         int    `json:"pid"`
	TimeCreated int64  `json:"timeCreated"`
	TimeUpdated int64  `json:"timeUpdated"`
	Size        *Size  `json:"size,omitempty"`
}

type Size struct {
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
	X    uint16 `json:"x"`
	Y    uint16 `json:"y"`
}

type PTYSession struct {
	Info     PTYInfo
	PTY      *os.File
	Command  *exec.Cmd
	mu       sync.Mutex
	closed   bool
	output   *ringBuffer
	listeners []chan []byte
}

type PTYStore struct {
	mu       sync.Mutex
	sessions map[string]*PTYSession
}

var GlobalPTYs = &PTYStore{
	sessions: make(map[string]*PTYSession),
}

func (s *PTYStore) Create(dir, title string, cmdArgs []string, size *Size) (*PTYSession, error) {
	id := "pty_" + uuid.New().String()[:12]

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	cmd := exec.Command(shell)
	if len(cmdArgs) > 0 {
		cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	}

	cmd.Dir = dir
	cmd.Env = os.Environ()

	var winSize *pty.Winsize
	if size != nil {
		winSize = &pty.Winsize{
			Rows: size.Rows,
			Cols: size.Cols,
			X:    size.X,
			Y:    size.Y,
		}
	}

	f, err := pty.StartWithSize(cmd, winSize)
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()
	session := &PTYSession{
		Info: PTYInfo{
			ID:          id,
			Title:       title,
			Directory:   dir,
			Command:     cmd.Path,
			Pid:         cmd.Process.Pid,
			TimeCreated: now,
			TimeUpdated: now,
			Size:        size,
		},
		PTY:     f,
		Command: cmd,
		output:  newRingBuffer(1024 * 1024), // 1MB ring buffer
	}

	// Start reading PTY output
	go session.readOutput()

	s.mu.Lock()
	s.sessions[id] = session
	s.mu.Unlock()

	return session, nil
}

func (s *PTYStore) Get(id string) *PTYSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[id]
}

func (s *PTYStore) List() []PTYInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []PTYInfo
	for _, sess := range s.sessions {
		result = append(result, sess.Info)
	}
	return result
}

func (s *PTYStore) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[id]; ok {
		sess.Close()
		delete(s.sessions, id)
	}
}

// ── Session operations ──────────────────────────────

func (ps *PTYSession) Write(data []byte) (int, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.closed {
		return 0, io.ErrClosedPipe
	}
	return ps.PTY.Write(data)
}

func (ps *PTYSession) Resize(size *Size) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.closed {
		return io.ErrClosedPipe
	}
	ws := &pty.Winsize{Rows: size.Rows, Cols: size.Cols, X: size.X, Y: size.Y}
	ps.Info.Size = size
	return pty.Setsize(ps.PTY, ws)
}

func (ps *PTYSession) Close() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.closed {
		return
	}
	ps.closed = true
	ps.PTY.Close()
	ps.Command.Process.Kill()

	// Close all listeners
	for _, ch := range ps.listeners {
		close(ch)
	}
	ps.listeners = nil
}

func (ps *PTYSession) Subscribe() chan []byte {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ch := make(chan []byte, 256)
	ps.listeners = append(ps.listeners, ch)

	// Replay buffered output
	replay := ps.output.Bytes()
	if len(replay) > 0 {
		// Send in chunks
		for i := 0; i < len(replay); i += 64 * 1024 {
			end := i + 64*1024
			if end > len(replay) {
				end = len(replay)
			}
			ch <- replay[i:end]
		}
	}

	return ch
}

func (ps *PTYSession) readOutput() {
	buf := make([]byte, 4096)
	for {
		n, err := ps.PTY.Read(buf)
		if err != nil {
			return
		}
		data := make([]byte, n)
		copy(data, buf[:n])

		ps.mu.Lock()
		ps.output.Write(data)
		ps.Info.TimeUpdated = time.Now().UnixMilli()
		listeners := append([]chan []byte{}, ps.listeners...)
		ps.mu.Unlock()

		for _, ch := range listeners {
			select {
			case ch <- data:
			default:
			}
		}
	}
}

// ── Ring Buffer ─────────────────────────────────────

type ringBuffer struct {
	buf  []byte
	size int
	pos  int
	full bool
	mu   sync.Mutex
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{
		buf:  make([]byte, size),
		size: size,
	}
}

func (r *ringBuffer) Write(data []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range data {
		r.buf[r.pos] = b
		r.pos++
		if r.pos >= r.size {
			r.pos = 0
			r.full = true
		}
	}
}

func (r *ringBuffer) Bytes() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.full {
		return r.buf[:r.pos]
	}
	result := make([]byte, r.size)
	copy(result, r.buf[r.pos:])
	copy(result[r.size-r.pos:], r.buf[:r.pos])
	return result
}

// ── Helpers ─────────────────────────────────────────

func NewPTYID() string {
	return "pty_" + uuid.New().String()[:12]
}

func PTYConnectToken(ptyID string) string {
	token := map[string]any{
		"ptyID": ptyID,
		"time":  time.Now().UnixMilli(),
	}
	b, _ := json.Marshal(token)
	return uuid.New().String() + "$" + string(b)
}
