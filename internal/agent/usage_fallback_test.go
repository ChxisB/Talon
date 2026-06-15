package agent

import (
	"errors"
	"testing"

	llm "github.com/ChxisB/talon/deps/llm"
	"github.com/ChxisB/talon/deps/testing/pkg/catwalk"
	"github.com/ChxisB/talon/internal/message"
	"github.com/ChxisB/talon/internal/session"
	"github.com/stretchr/testify/require"
)

func TestUsageIsZero(t *testing.T) {
	t.Parallel()

	require.True(t, usageIsZero(llm.Usage{}))
	require.False(t, usageIsZero(llm.Usage{InputTokens: 1}))
	require.False(t, usageIsZero(llm.Usage{OutputTokens: 1}))
	require.False(t, usageIsZero(llm.Usage{TotalTokens: 1}))
	require.False(t, usageIsZero(llm.Usage{ReasoningTokens: 1}))
	require.False(t, usageIsZero(llm.Usage{CacheCreationTokens: 1}))
	require.False(t, usageIsZero(llm.Usage{CacheReadTokens: 1}))
}

func TestFallbackStepUsageKeepsProviderUsage(t *testing.T) {
	t.Parallel()

	usage := llm.Usage{
		InputTokens:  10,
		OutputTokens: 5,
		TotalTokens:  15,
	}
	step := llm.StepResult{
		Response: llm.Response{Usage: usage},
	}

	fallbackUsage, estimated := fallbackStepUsage(nil, step)
	require.False(t, estimated)
	require.Equal(t, usage, fallbackUsage)
}

func TestFallbackStepUsageEstimatesPromptAndAssistantText(t *testing.T) {
	t.Parallel()

	messages := []llm.Message{
		llm.NewUserMessage("please explain the implementation details"),
	}
	step := llm.StepResult{
		Response: llm.Response{
			Content: llm.ResponseContent{
				llm.TextContent{Text: "the implementation stores state safely"},
			},
		},
	}

	usage, estimated := fallbackStepUsage(messages, step)
	require.True(t, estimated)
	require.Positive(t, usage.InputTokens)
	require.Positive(t, usage.OutputTokens)
	require.Equal(t, usage.InputTokens+usage.OutputTokens, usage.TotalTokens)
}

func TestFallbackStepUsageEstimatesReasoning(t *testing.T) {
	t.Parallel()

	messages := []llm.Message{
		{
			Role: llm.MessageRoleAssistant,
			Content: []llm.MessagePart{
				llm.ReasoningPart{Text: "first reason about the request"},
			},
		},
	}
	step := llm.StepResult{
		Response: llm.Response{
			Content: llm.ResponseContent{
				llm.ReasoningContent{Text: "second reason about the answer"},
			},
		},
	}

	usage, estimated := fallbackStepUsage(messages, step)
	require.True(t, estimated)
	require.Positive(t, usage.InputTokens)
	require.Positive(t, usage.OutputTokens)
}

func TestFallbackStepUsageEstimatesToolCalls(t *testing.T) {
	t.Parallel()

	step := llm.StepResult{
		Response: llm.Response{
			Content: llm.ResponseContent{
				llm.ToolCallContent{
					ToolCallID: "tool-call-1",
					ToolName:   "view",
					Input:      `{"file_path":"/tmp/example.go"}`,
				},
			},
		},
	}

	usage, estimated := fallbackStepUsage(nil, step)
	require.True(t, estimated)
	require.Zero(t, usage.InputTokens)
	require.Positive(t, usage.OutputTokens)
	require.Equal(t, usage.OutputTokens, usage.TotalTokens)
}

func TestFallbackStepUsageEstimatesToolResults(t *testing.T) {
	t.Parallel()

	messages := []llm.Message{
		{
			Role: llm.MessageRoleTool,
			Content: []llm.MessagePart{
				llm.ToolResultPart{
					ToolCallID: "tool-call-1",
					Output: llm.ToolResultOutputContentText{
						Text: "file contents returned by the tool",
					},
				},
				llm.ToolResultPart{
					ToolCallID: "tool-call-2",
					Output: llm.ToolResultOutputContentError{
						Error: errors.New("permission denied"),
					},
				},
				llm.ToolResultPart{
					ToolCallID: "tool-call-3",
					Output: llm.ToolResultOutputContentMedia{
						MediaType: "image/png",
						Text:      "screenshot",
						Data:      "abc123",
					},
				},
			},
		},
	}

	usage, estimated := fallbackStepUsage(messages, llm.StepResult{})
	require.True(t, estimated)
	require.Positive(t, usage.InputTokens)
	require.Zero(t, usage.OutputTokens)
	require.Equal(t, usage.InputTokens, usage.TotalTokens)
}

func TestFallbackStepUsageSkipsClientToolResultsAsOutput(t *testing.T) {
	t.Parallel()

	step := llm.StepResult{
		Response: llm.Response{
			Content: llm.ResponseContent{
				llm.ToolResultContent{
					ToolCallID: "tool-call-1",
					ToolName:   "bash",
					Result: llm.ToolResultOutputContentText{
						Text: "large client-executed payload that should not count as model output tokens",
					},
				},
			},
		},
	}

	usage, estimated := fallbackStepUsage(nil, step)
	require.False(t, estimated)
	require.Zero(t, usage.OutputTokens)
}

func TestFallbackStepUsageCountsProviderToolResultsAsOutput(t *testing.T) {
	t.Parallel()

	step := llm.StepResult{
		Response: llm.Response{
			Content: llm.ResponseContent{
				llm.ToolResultContent{
					ToolCallID:       "tool-call-1",
					ToolName:         "web_search",
					ProviderExecuted: true,
					ClientMetadata:   "provider metadata",
					Result:           llm.ToolResultOutputContentText{Text: "provider-executed result"},
				},
			},
		},
	}

	usage, estimated := fallbackStepUsage(nil, step)
	require.True(t, estimated)
	require.Positive(t, usage.OutputTokens)
	require.Equal(t, usage.OutputTokens, usage.TotalTokens)
}

func TestFallbackStepUsageReturnsZeroWithoutContent(t *testing.T) {
	t.Parallel()

	usage, estimated := fallbackStepUsage(nil, llm.StepResult{})
	require.False(t, estimated)
	require.True(t, usageIsZero(usage))
}

func TestUpdateSessionUsageSkipsEstimatedCost(t *testing.T) {
	t.Parallel()

	agent := &sessionAgent{}
	currentSession := &session.Session{ID: "session-id", Cost: 1.25}
	model := Model{CatwalkCfg: catwalk.Model{CostPer1MIn: 10, CostPer1MOut: 20}}
	usage := llm.Usage{InputTokens: 1000, OutputTokens: 2000}

	agent.updateSessionUsage(model, currentSession, usage, nil, true)

	require.Equal(t, 1.25, currentSession.Cost)
	require.Equal(t, int64(1000), currentSession.PromptTokens)
	require.Equal(t, int64(2000), currentSession.CompletionTokens)
	require.True(t, currentSession.EstimatedUsage)
}

func TestUpdateSessionUsageKeepsCountersForZeroUsage(t *testing.T) {
	t.Parallel()

	agent := &sessionAgent{}
	currentSession := &session.Session{
		ID:               "session-id",
		PromptTokens:     123,
		CompletionTokens: 456,
		Cost:             1.25,
	}
	model := Model{CatwalkCfg: catwalk.Model{CostPer1MIn: 10, CostPer1MOut: 20}}

	agent.updateSessionUsage(model, currentSession, llm.Usage{}, nil, false)

	require.Equal(t, 1.25, currentSession.Cost)
	require.Equal(t, int64(123), currentSession.PromptTokens)
	require.Equal(t, int64(456), currentSession.CompletionTokens)
}

func TestUpdateSessionUsagePreservesOmittedCountersForPartialUsage(t *testing.T) {
	t.Parallel()

	agent := &sessionAgent{}
	currentSession := &session.Session{
		ID:               "session-id",
		PromptTokens:     123,
		CompletionTokens: 456,
	}
	model := Model{CatwalkCfg: catwalk.Model{CostPer1MIn: 10, CostPer1MOut: 20}}
	usage := llm.Usage{InputTokens: 789}

	agent.updateSessionUsage(model, currentSession, usage, nil, false)

	require.Equal(t, int64(789), currentSession.PromptTokens)
	require.Equal(t, int64(456), currentSession.CompletionTokens)
}

func TestUpdateSessionUsagePreservesCountersForTotalOnlyUsage(t *testing.T) {
	t.Parallel()

	agent := &sessionAgent{}
	currentSession := &session.Session{
		ID:               "session-id",
		PromptTokens:     123,
		CompletionTokens: 456,
	}
	model := Model{CatwalkCfg: catwalk.Model{CostPer1MIn: 10, CostPer1MOut: 20}}
	usage := llm.Usage{TotalTokens: 100}

	agent.updateSessionUsage(model, currentSession, usage, nil, false)

	require.Equal(t, int64(123), currentSession.PromptTokens)
	require.Equal(t, int64(456), currentSession.CompletionTokens)
}

func TestUpdateSessionUsagePreservesPromptForOutputOnlyUsage(t *testing.T) {
	t.Parallel()

	agent := &sessionAgent{}
	currentSession := &session.Session{
		ID:               "session-id",
		PromptTokens:     123,
		CompletionTokens: 456,
	}
	model := Model{CatwalkCfg: catwalk.Model{CostPer1MIn: 10, CostPer1MOut: 20}}
	usage := llm.Usage{OutputTokens: 50}

	agent.updateSessionUsage(model, currentSession, usage, nil, false)

	require.Equal(t, int64(123), currentSession.PromptTokens)
	require.Equal(t, int64(50), currentSession.CompletionTokens)
}

func TestUpdateSessionUsageKeepsCountersForEstimatedZeroUsage(t *testing.T) {
	t.Parallel()

	agent := &sessionAgent{}
	currentSession := &session.Session{
		ID:               "session-id",
		PromptTokens:     123,
		CompletionTokens: 456,
		Cost:             1.25,
	}
	model := Model{CatwalkCfg: catwalk.Model{CostPer1MIn: 10, CostPer1MOut: 20}}

	agent.updateSessionUsage(model, currentSession, llm.Usage{}, nil, true)

	require.Equal(t, 1.25, currentSession.Cost)
	require.Equal(t, int64(123), currentSession.PromptTokens)
	require.Equal(t, int64(456), currentSession.CompletionTokens)
}

func TestSummaryCompletionTokens(t *testing.T) {
	t.Parallel()

	summaryMessage := message.Message{
		Parts: []message.ContentPart{
			message.TextContent{Text: "summary text"},
			message.ReasoningContent{Thinking: "reasoning text"},
		},
	}

	require.Equal(t, int64(42), summaryCompletionTokens(llm.Usage{OutputTokens: 42}, summaryMessage))
	require.Equal(t, approxTokenCount("summary text")+approxTokenCount("reasoning text"), summaryCompletionTokens(llm.Usage{}, summaryMessage))
	require.Zero(t, summaryCompletionTokens(llm.Usage{}, message.Message{}))
}

func TestUpdateSessionUsageAddsProviderCost(t *testing.T) {
	t.Parallel()

	agent := &sessionAgent{}
	currentSession := &session.Session{ID: "session-id", Cost: 1.25}
	model := Model{CatwalkCfg: catwalk.Model{CostPer1MIn: 10, CostPer1MOut: 20}}
	usage := llm.Usage{InputTokens: 1000, OutputTokens: 2000}

	agent.updateSessionUsage(model, currentSession, usage, nil, false)

	require.Equal(t, 1.3, currentSession.Cost)
	require.Equal(t, int64(1000), currentSession.PromptTokens)
	require.Equal(t, int64(2000), currentSession.CompletionTokens)
	require.False(t, currentSession.EstimatedUsage)
}
