Capture a screenshot of the current screen.

Use this tool when you need to see what's currently displayed on the user's screen. This is useful for understanding the user's visual context.

Parameters:
- `question` (optional): A specific question about the screenshot. If provided, the screenshot will be analyzed automatically and the answer returned directly.

If a local vision model (like MiniCPM-V via Ollama) is configured, the screenshot will be analyzed and you'll receive a text description of what's on the screen. Otherwise, the raw image is returned for models that support image inputs natively.

Use cases:
- Understanding what the user is looking at
- Analyzing UI layouts, error messages, or visual content on screen
- Getting visual context when the user asks about something they're seeing
