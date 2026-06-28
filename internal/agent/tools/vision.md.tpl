Analyze an image or document file using a local vision model.

Use this tool when you need to understand the contents of an image, screenshot, diagram, chart, or document. This is especially useful when your primary model does not natively support image inputs.

The tool sends the image to a local vision model (like MiniCPM-V running via Ollama) and returns a text description of what's in the image.

Parameters:
- `image_path` (required): The path to the image or document file to analyze.
- `question` (optional): A specific question or instruction about the image. If omitted, the model will provide a general description.

Use cases:
- Analyzing screenshots or screen captures
- Reading text from images (OCR-like tasks)
- Describing charts, diagrams, or graphs
- Identifying objects, people, or scenes in photos
- Processing document scans or PDF pages
