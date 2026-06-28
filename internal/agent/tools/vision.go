package tools

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	llm "github.com/ChxisB/talon/deps/llm"
)

// VisionToolName is the name of the vision analysis tool.
const VisionToolName = "analyze_image"

// DefaultVisionEndpoint is the default Ollama OpenAI-compatible endpoint.
const DefaultVisionEndpoint = "http://localhost:11434/v1/chat/completions"

// DefaultVisionModel is the default vision model name.
const DefaultVisionModel = "minicpm-v"

// MaxImageSize is the maximum image file size we'll send (20MB).
const MaxImageSize = 20 * 1024 * 1024

//go:embed vision.md.tpl
var visionDescriptionTmpl []byte

var visionDescriptionTpl = template.Must(
	template.New("visionDescription").Parse(string(visionDescriptionTmpl)),
)

// VisionParams is the input for the analyze_image tool.
type VisionParams struct {
	ImagePath string `json:"image_path" description:"The path to the image or document file to analyze"`
	Question  string `json:"question,omitempty" description:"An optional question or instruction about the image"`
}

// VisionConfig holds configuration for the vision tool.
type VisionConfig struct {
	// Endpoint is the OpenAI-compatible API endpoint for the vision model.
	Endpoint string
	// Model is the vision model name to use.
	Model string
	// Timeout is the HTTP client timeout for requests.
	Timeout time.Duration
}

// visionRequest is the OpenAI chat completions request format.
type visionRequest struct {
	Model    string            `json:"model"`
	Messages []visionMessage   `json:"messages"`
	Stream   bool              `json:"stream"`
}

type visionMessage struct {
	Role    string          `json:"role"`
	Content []visionContent `json:"content"`
}

type visionContent struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	ImageURL *visionImageURL `json:"image_url,omitempty"`
}

type visionImageURL struct {
	URL string `json:"url"`
}

// visionResponse is the OpenAI chat completions response format.
type visionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// DefaultVisionConfig returns a VisionConfig with sensible defaults.
func DefaultVisionConfig() VisionConfig {
	return VisionConfig{
		Endpoint: DefaultVisionEndpoint,
		Model:    DefaultVisionModel,
		Timeout:  120 * time.Second,
	}
}

func visionDescription() string {
	return renderTemplate(visionDescriptionTpl, nil)
}

// NewAnalyzeImageTool creates a tool that sends an image to a local vision model
// (like MiniCPM-V via Ollama) for analysis and returns the text description.
func NewAnalyzeImageTool(cfg VisionConfig) llm.AgentTool {
	return llm.NewAgentTool(
		VisionToolName,
		visionDescription(),
		func(ctx context.Context, params VisionParams, call llm.ToolCall) (llm.ToolResponse, error) {
			if params.ImagePath == "" {
				return llm.NewTextErrorResponse("The image_path parameter is required."), nil
			}

			// Read the image file.
			imageData, err := os.ReadFile(params.ImagePath)
			if err != nil {
				return llm.NewTextErrorResponse(
					fmt.Sprintf("Failed to read image file %q: %s", params.ImagePath, err),
				), nil
			}

			if len(imageData) > MaxImageSize {
				return llm.NewTextErrorResponse(
					fmt.Sprintf("Image file is too large (%d bytes). Maximum size is %d bytes.", len(imageData), MaxImageSize),
				), nil
			}

			// Detect MIME type from content.
			mimeType := detectMimeType(imageData, params.ImagePath)

			// Build the question text.
			question := params.Question
			if question == "" {
				question = "Please describe this image in detail, including any text, objects, people, or important visual elements."
			}

			// Base64 encode the image.
			encoded := base64.StdEncoding.EncodeToString(imageData)
			dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)

			// Build the request.
			reqBody := visionRequest{
				Model: cfg.Model,
				Messages: []visionMessage{
					{
						Role: "user",
						Content: []visionContent{
							{
								Type: "text",
								Text: question,
							},
							{
								Type:     "image_url",
								ImageURL: &visionImageURL{URL: dataURL},
							},
						},
					},
				},
				Stream: false,
			}

			bodyBytes, err := json.Marshal(reqBody)
			if err != nil {
				return llm.NewTextErrorResponse(fmt.Sprintf("Failed to encode request: %s", err)), nil
			}

			// Make the HTTP request.
			httpClient := &http.Client{Timeout: cfg.Timeout}
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Endpoint, bytes.NewReader(bodyBytes))
			if err != nil {
				return llm.NewTextErrorResponse(fmt.Sprintf("Failed to create request: %s", err)), nil
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := httpClient.Do(req)
			if err != nil {
				return llm.NewTextErrorResponse(
					fmt.Sprintf("Failed to connect to vision model at %s: %s. Make sure Ollama is running with the %s model loaded.", cfg.Endpoint, err, cfg.Model),
				), nil
			}
			defer resp.Body.Close()

			respBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return llm.NewTextErrorResponse(fmt.Sprintf("Failed to read response: %s", err)), nil
			}

			if resp.StatusCode != http.StatusOK {
				return llm.NewTextErrorResponse(
					fmt.Sprintf("Vision model returned HTTP %d: %s", resp.StatusCode, string(respBytes)),
				), nil
			}

			var result visionResponse
			if err := json.Unmarshal(respBytes, &result); err != nil {
				return llm.NewTextErrorResponse(fmt.Sprintf("Failed to parse response: %s", err)), nil
			}

			if result.Error != nil && result.Error.Message != "" {
				return llm.NewTextErrorResponse(
					fmt.Sprintf("Vision model error: %s", result.Error.Message),
				), nil
			}

			if len(result.Choices) == 0 {
				return llm.NewTextErrorResponse("Vision model returned no response."), nil
			}

			analysis := strings.TrimSpace(result.Choices[0].Message.Content)
			if analysis == "" {
				return llm.NewTextErrorResponse("Vision model returned an empty response."), nil
			}

			return llm.NewTextResponse(analysis), nil
		},
	)
}

// detectMimeType sniffs the MIME type from the image data, falling back to
// the file extension.
func detectMimeType(data []byte, filePath string) string {
	// Try to detect from content first.
	detected := http.DetectContentType(data)
	if strings.HasPrefix(detected, "image/") {
		return detected
	}

	// Fall back to extension-based detection.
	ext := strings.ToLower(filePath)
	switch {
	case strings.HasSuffix(ext, ".jpg"), strings.HasSuffix(ext, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(ext, ".png"):
		return "image/png"
	case strings.HasSuffix(ext, ".gif"):
		return "image/gif"
	case strings.HasSuffix(ext, ".webp"):
		return "image/webp"
	case strings.HasSuffix(ext, ".pdf"):
		return "application/pdf"
	default:
		return "image/png"
	}
}
