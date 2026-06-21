package service

import (
	"sync"

	"github.com/google/uuid"
)

// ── Types ───────────────────────────────────────────

type QuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

type QuestionInfo struct {
	Question string           `json:"question"`
	Header   string           `json:"header"`
	Options  []QuestionOption `json:"options"`
	Multiple bool             `json:"multiple"`
	Custom   bool             `json:"custom"`
}

type QuestionRequest struct {
	ID          string         `json:"id"`
	SessionID   string         `json:"sessionID"`
	Questions   []QuestionInfo `json:"questions"`
	Tool        *QuestionTool  `json:"tool,omitempty"`
	TimeCreated int64          `json:"timeCreated"`
}

type QuestionTool struct {
	MessageID string `json:"messageID"`
	CallID    string `json:"callID"`
}

type QuestionReply struct {
	RequestID string     `json:"requestID"`
	Answers   [][]string `json:"answers"`
}

// ── Store ───────────────────────────────────────────

type QuestionStore struct {
	mu      sync.RWMutex
	pending map[string]*QuestionRequest
}

var GlobalQuestions = &QuestionStore{
	pending: make(map[string]*QuestionRequest),
}

func (s *QuestionStore) List(location string) []*QuestionRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*QuestionRequest
	for _, q := range s.pending {
		result = append(result, q)
	}
	return result
}

func (s *QuestionStore) ListBySession(sessionID string) []*QuestionRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*QuestionRequest
	for _, q := range s.pending {
		if q.SessionID == sessionID {
			result = append(result, q)
		}
	}
	return result
}

func (s *QuestionStore) Add(q *QuestionRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending[q.ID] = q
}

func (s *QuestionStore) Get(id string) *QuestionRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pending[id]
}

func (s *QuestionStore) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pending, id)
}

func NewQuestionID() string {
	return "que_" + uuid.New().String()[:12]
}
