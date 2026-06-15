package agent

import (
	"fmt"

	llm "github.com/ChxisB/talon/deps/llm"
)

func usageIsZero(usage llm.Usage) bool {
	return usage.InputTokens == 0 &&
		usage.OutputTokens == 0 &&
		usage.TotalTokens == 0 &&
		usage.ReasoningTokens == 0 &&
		usage.CacheCreationTokens == 0 &&
		usage.CacheReadTokens == 0
}

func fallbackStepUsage(messages []llm.Message, step llm.StepResult) (llm.Usage, bool) {
	if !usageIsZero(step.Usage) {
		return step.Usage, false
	}

	inputTokens := estimateMessageTokens(messages)
	outputTokens := estimateStepCompletionTokens(step)
	if inputTokens == 0 && outputTokens == 0 {
		return llm.Usage{}, false
	}

	return llm.Usage{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
	}, true
}

func cloneFantasyMessages(messages []llm.Message) []llm.Message {
	cloned := make([]llm.Message, len(messages))
	for i, msg := range messages {
		cloned[i] = msg
		cloned[i].Content = append([]llm.MessagePart(nil), msg.Content...)
	}
	return cloned
}

func estimateMessageTokens(messages []llm.Message) int64 {
	var tokens int64
	for _, msg := range messages {
		tokens += approxTokenCount(string(msg.Role))
		for _, part := range msg.Content {
			tokens += estimateMessagePartTokens(part)
		}
	}
	return tokens
}

func estimateStepCompletionTokens(step llm.StepResult) int64 {
	var tokens int64
	for _, content := range step.Content {
		switch c := content.(type) {
		case llm.TextContent:
			tokens += approxTokenCount(c.Text)
		case *llm.TextContent:
			tokens += approxTokenCount(c.Text)
		case llm.ReasoningContent:
			tokens += approxTokenCount(c.Text)
		case *llm.ReasoningContent:
			tokens += approxTokenCount(c.Text)
		case llm.FileContent:
			tokens += estimateGeneratedFileTokens(c)
		case *llm.FileContent:
			tokens += estimateGeneratedFileTokens(*c)
		case llm.SourceContent:
			tokens += estimateSourceTokens(c)
		case *llm.SourceContent:
			tokens += estimateSourceTokens(*c)
		case llm.ToolCallContent:
			tokens += estimateToolCallTokens(c.ToolName, c.Input)
		case *llm.ToolCallContent:
			tokens += estimateToolCallTokens(c.ToolName, c.Input)
		case llm.ToolResultContent:
			if c.ProviderExecuted {
				tokens += estimateToolResultContentTokens(c.ToolCallID, c.ToolName, c.ClientMetadata, c.Result)
			}
		case *llm.ToolResultContent:
			if c.ProviderExecuted {
				tokens += estimateToolResultContentTokens(c.ToolCallID, c.ToolName, c.ClientMetadata, c.Result)
			}
		}
	}
	return tokens
}

func estimateMessagePartTokens(part llm.MessagePart) int64 {
	switch p := part.(type) {
	case llm.TextPart:
		return approxTokenCount(p.Text)
	case *llm.TextPart:
		return approxTokenCount(p.Text)
	case llm.ReasoningPart:
		return approxTokenCount(p.Text)
	case *llm.ReasoningPart:
		return approxTokenCount(p.Text)
	case llm.FilePart:
		return estimateFilePartTokens(p)
	case *llm.FilePart:
		return estimateFilePartTokens(*p)
	case llm.ToolCallPart:
		return estimateToolCallTokens(p.ToolName, p.Input)
	case *llm.ToolCallPart:
		return estimateToolCallTokens(p.ToolName, p.Input)
	case llm.ToolResultPart:
		return estimateToolResultContentTokens(p.ToolCallID, "", "", p.Output)
	case *llm.ToolResultPart:
		return estimateToolResultContentTokens(p.ToolCallID, "", "", p.Output)
	default:
		return 0
	}
}

func estimateToolCallTokens(toolName, input string) int64 {
	return approxTokenCount(toolName) + approxTokenCount(input)
}

func estimateToolResultContentTokens(toolCallID, toolName, metadata string, output llm.ToolResultOutputContent) int64 {
	tokens := approxTokenCount(toolCallID) + approxTokenCount(toolName) + approxTokenCount(metadata)
	switch result := output.(type) {
	case llm.ToolResultOutputContentText:
		tokens += approxTokenCount(result.Text)
	case *llm.ToolResultOutputContentText:
		tokens += approxTokenCount(result.Text)
	case llm.ToolResultOutputContentError:
		if result.Error != nil {
			tokens += approxTokenCount(result.Error.Error())
		}
	case *llm.ToolResultOutputContentError:
		if result.Error != nil {
			tokens += approxTokenCount(result.Error.Error())
		}
	case llm.ToolResultOutputContentMedia:
		tokens += estimateMediaTokens(result.MediaType, result.Text, len(result.Data))
	case *llm.ToolResultOutputContentMedia:
		tokens += estimateMediaTokens(result.MediaType, result.Text, len(result.Data))
	}
	return tokens
}

func estimateFilePartTokens(file llm.FilePart) int64 {
	return estimateMediaTokens(file.MediaType, file.Filename, len(file.Data))
}

func estimateGeneratedFileTokens(file llm.FileContent) int64 {
	return estimateMediaTokens(file.MediaType, "", len(file.Data))
}

func estimateMediaTokens(mediaType, text string, dataBytes int) int64 {
	if dataBytes == 0 {
		return approxTokenCount(mediaType) + approxTokenCount(text)
	}
	return approxTokenCount(fmt.Sprintf("%s %s %d bytes", mediaType, text, dataBytes))
}

func estimateSourceTokens(source llm.SourceContent) int64 {
	return approxTokenCount(string(source.SourceType)) +
		approxTokenCount(source.ID) +
		approxTokenCount(source.URL) +
		approxTokenCount(source.Title) +
		approxTokenCount(source.MediaType) +
		approxTokenCount(source.Filename)
}

func approxTokenCount(s string) int64 {
	if s == "" {
		return 0
	}
	return int64((len(s) + 3) / 4)
}
