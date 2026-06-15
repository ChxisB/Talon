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
	"os/exec"
	"strings"
	"time"

	llm "github.com/ChxisB/talon/deps/llm"
)

// ScreenshotToolName is the name of the screenshot tool.
const ScreenshotToolName = "screenshot"

//go:embed screenshot.md.tpl
var screenshotDescriptionTmpl []byte

var screenshotDescriptionTpl = template.Must(
	template.New("screenshotDescription").Parse(string(screenshotDescriptionTmpl)),
)

// ScreenshotParams is the input for the screenshot tool.
type ScreenshotParams struct {
	Question string `json:"question,omitempty" description:"An optional question about the screenshot. If provided, the screenshot will be analyzed and the answer returned directly."`
}

// ScreenshotConfig holds configuration for the screenshot tool.
type ScreenshotConfig struct {
	// VisionEndpoint is the optional OpenAI-compatible API endpoint for a vision model.
	// When set, screenshots are automatically analyzed using this vision model.
	// When empty, the raw image is returned for natively vision-capable models.
	VisionEndpoint string
	// VisionModel is the vision model name to use for auto-analysis.
	VisionModel string
	// VisionTimeout is the timeout for vision model requests.
	VisionTimeout time.Duration
	// TempDir is the directory for temporary screenshot files.
	// Empty means use os.TempDir().
	TempDir string
}

// DefaultScreenshotConfig returns a ScreenshotConfig with sensible defaults.
func DefaultScreenshotConfig() ScreenshotConfig {
	return ScreenshotConfig{
		VisionModel:   DefaultVisionModel,
		VisionTimeout: 120 * time.Second,
	}
}

func screenshotDescription() string {
	return renderTemplate(screenshotDescriptionTpl, nil)
}

// captureScreen captures the screen to a temporary PNG file and returns
// the file path and any error.
func captureScreen(tempDir string) (string, error) {
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	// Create temp file with .png extension.
	tmpFile, err := os.CreateTemp(tempDir, "talon-screenshot-*.png")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	screenshotPath := tmpFile.Name()
	tmpFile.Close()

	// Platform-specific screenshot capture.
	switch {
	case isCommandAvailable("screencapture"):
		// macOS.
		cmd := exec.Command("screencapture", "-x", "-t", "png", screenshotPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			os.Remove(screenshotPath)
			return "", fmt.Errorf("screencapture failed: %s: %s", err, string(output))
		}
	case isCommandAvailable("import"):
		// Linux with ImageMagick.
		cmd := exec.Command("import", "-silent", screenshotPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			os.Remove(screenshotPath)
			return "", fmt.Errorf("import (ImageMagick) failed: %s: %s", err, string(output))
		}
	case isCommandAvailable("gnome-screenshot"):
		// Linux with GNOME.
		cmd := exec.Command("gnome-screenshot", "-f", screenshotPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			os.Remove(screenshotPath)
			return "", fmt.Errorf("gnome-screenshot failed: %s: %s", err, string(output))
		}
	default:
		os.Remove(screenshotPath)
		return "", fmt.Errorf("no screenshot tool found. Install screencapture (macOS), import (ImageMagick), or gnome-screenshot (Linux)")
	}

	// Verify the file was created and has content.
	info, err := os.Stat(screenshotPath)
	if err != nil || info.Size() == 0 {
		os.Remove(screenshotPath)
		return "", fmt.Errorf("screenshot file is empty or missing")
	}

	return screenshotPath, nil
}

// isCommandAvailable checks if a command is available on the system.
func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// analyzeImageWithVision sends an image to the vision model and returns the analysis.
func analyzeImageWithVision(ctx context.Context, imagePath, question, endpoint, model string, timeout time.Duration) (string, error) {
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
	}

	mimeType := detectMimeType(imageData, imagePath)
	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)

	if question == "" {
		question = "Please describe this screenshot in detail, including any text, UI elements, code, or important visual information visible on the screen."
	}

	reqBody := visionRequest{
		Model: model,
		Messages: []visionMessage{
			{
				Role: "user",
				Content: []visionContent{
					{Type: "text", Text: question},
					{Type: "image_url", ImageURL: &visionImageURL{URL: dataURL}},
				},
			},
		},
		Stream: false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to encode request: %w", err)
	}

	httpClient := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to vision model at %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vision model returned HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var result visionResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != nil && result.Error.Message != "" {
		return "", fmt.Errorf("vision model error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("vision model returned no response")
	}

	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

// NewScreenshotTool creates a tool that captures the screen and optionally
// analyzes it using a local vision model (like MiniCPM-V).
func NewScreenshotTool(cfg ScreenshotConfig) llm.AgentTool {
	return llm.NewAgentTool(
		ScreenshotToolName,
		screenshotDescription(),
		func(ctx context.Context, params ScreenshotParams, call llm.ToolCall) (llm.ToolResponse, error) {
			// Capture the screen.
			screenshotPath, err := captureScreen(cfg.TempDir)
			if err != nil {
				return llm.NewTextErrorResponse(
					fmt.Sprintf("Failed to capture screenshot: %s. Make sure you have a screenshot tool installed.", err),
				), nil
			}

			// Read the screenshot data.
			imageData, err := os.ReadFile(screenshotPath)
			if err != nil {
				os.Remove(screenshotPath)
				return llm.NewTextErrorResponse(fmt.Sprintf("Failed to read screenshot: %s", err)), nil
			}

			// If a vision endpoint is configured, analyze the screenshot
			// automatically and return the text analysis.
			if cfg.VisionEndpoint != "" {
				question := params.Question
				if question == "" && params.Question == "" {
					question = "Please describe this screenshot in detail."
				}

				analysis, err := analyzeImageWithVision(
					ctx, screenshotPath, question,
					cfg.VisionEndpoint, cfg.VisionModel, cfg.VisionTimeout,
				)
				// Clean up the temp file.
				os.Remove(screenshotPath)

				if err != nil {
					return llm.NewTextErrorResponse(
						fmt.Sprintf("Screenshot captured but vision analysis failed: %s", err),
					), nil
				}

				// Return the analysis with metadata about the image for rendering.
				return llm.WithResponseMetadata(
					llm.NewTextResponse(analysis),
					fmt.Sprintf(`{"screenshot":true,"file":"%s"}`, screenshotPath),
				), nil
			}

			// No vision endpoint: return image data for vision-capable models.
			// Clean up the temp file after reading.
			os.Remove(screenshotPath)
			return llm.NewImageResponse(imageData, "image/png"), nil
		},
	)
}
