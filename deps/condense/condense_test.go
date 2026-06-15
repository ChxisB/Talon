package condense

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestDetectContent_JSONArray(t *testing.T) {
	content := `[{"id": 1, "name": "foo"}, {"id": 2, "name": "bar"}]`
	r := DetectContent(content)
	if r.ContentType != ContentTypeJSONArray {
		t.Fatalf("expected JSON_ARRAY, got %s", r.ContentType)
	}
	if r.Confidence != 1.0 {
		t.Fatalf("expected confidence 1.0, got %f", r.Confidence)
	}
	if !IsJSONArrayOfDicts(content) {
		t.Fatal("expected is_dict_array=true")
	}
}

func TestDetectContent_NonDictArray(t *testing.T) {
	content := `["a", "b", "c"]`
	r := DetectContent(content)
	if r.ContentType != ContentTypeJSONArray {
		t.Fatalf("expected JSON_ARRAY, got %s", r.ContentType)
	}
	if r.Confidence != 0.8 {
		t.Fatalf("expected confidence 0.8, got %f", r.Confidence)
	}
}

func TestDetectContent_Code(t *testing.T) {
	content := `
package main

func main() {
	fmt.Println("hello")
}`
	r := DetectContent(content)
	if r.ContentType != ContentTypeSourceCode {
		t.Fatalf("expected SOURCE_CODE, got %s", r.ContentType)
	}
}

func TestDetectContent_Python(t *testing.T) {
	content := `def hello():
    print("world")

class Test:
    pass`
	r := DetectContent(content)
	if r.ContentType != ContentTypeSourceCode {
		t.Fatalf("expected SOURCE_CODE, got %s", r.ContentType)
	}
	lang, _ := r.Metadata["language"].(string)
	if lang != "python" {
		t.Fatalf("expected language python, got %s", lang)
	}
}

func TestDetectContent_SearchResults(t *testing.T) {
	content := `src/main.go:42:func main() {
src/utils.go:10:func helper() {
src/main.go:45:	fmt.Println("hello")
src/cli.go:5:package main`
	r := DetectContent(content)
	if r.ContentType != ContentTypeSearchResults {
		t.Fatalf("expected SEARCH_RESULTS, got %s", r.ContentType)
	}
}

func TestDetectContent_BuildOutput(t *testing.T) {
	content := `ERROR: build failed
WARNING: deprecated function
INFO: processing...
FAILED: test_main
Traceback (most recent call last):
  File "test.py", line 10, in <module>`
	r := DetectContent(content)
	if r.ContentType != ContentTypeBuildOutput {
		t.Fatalf("expected BUILD_OUTPUT, got %s", r.ContentType)
	}
}

func TestDetectContent_GitDiff(t *testing.T) {
	content := `diff --git a/main.go b/main.go
--- a/main.go
@@ -10,6 +10,7 @@
+func newFunc() {
`
	r := DetectContent(content)
	if r.ContentType != ContentTypeGitDiff {
		t.Fatalf("expected GIT_DIFF, got %s", r.ContentType)
	}
}

func TestDetectContent_HTML(t *testing.T) {
	content := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><div>Hello</div></body>
</html>`
	r := DetectContent(content)
	if r.ContentType != ContentTypeHTML {
		t.Fatalf("expected HTML, got %s", r.ContentType)
	}
}

func TestDetectContent_PlainText(t *testing.T) {
	content := `Hello, this is just some plain text.`
	r := DetectContent(content)
	if r.ContentType != ContentTypePlainText {
		t.Fatalf("expected PLAIN_TEXT, got %s", r.ContentType)
	}
}

func TestCompressContent_TooShort(t *testing.T) {
	content := "hi"
	compressed, modified, _ := CompressContent(content, DefaultConfig())
	if modified {
		t.Fatal("should not modify short content")
	}
	if compressed != content {
		t.Fatal("should return original")
	}
}

func TestCrushJSONArray_Dedup(t *testing.T) {
	items := make([]map[string]any, 50)
	for i := 0; i < 50; i++ {
		items[i] = map[string]any{"id": float64(i), "name": "tool-item-" + fmt.Sprint(i), "value": float64(i % 3), "status": "ok"}
	}
	// Make many duplicates
	for i := 10; i < 30; i++ {
		items[i] = items[i%10]
	}

	data, _ := json.Marshal(items)
	result := CrushJSONArray(string(data), DefaultCrusherConfig())

	if !result.WasModified {
		t.Fatal("expected modification")
	}
	if result.ItemsAfter >= result.ItemsBefore {
		t.Fatal("expected fewer items after crush")
	}

	// Verify output is valid JSON
	var parsed []any
	if err := json.Unmarshal([]byte(result.Compressed), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	t.Logf("crushed %d -> %d items (ratio: %.2f)", result.ItemsBefore, result.ItemsAfter, result.SavingsRatio)
}

func TestCrushJSONArray_SmallArray(t *testing.T) {
	items := []map[string]any{
		{"id": float64(1), "name": "a"},
		{"id": float64(2), "name": "b"},
	}
	data, _ := json.Marshal(items)
	result := CrushJSONArray(string(data), DefaultCrusherConfig())

	if result.WasModified {
		t.Fatal("small array should not be crushed")
	}
}

func TestCrushJSONArray_NonObjectArray(t *testing.T) {
	items := []any{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	data, _ := json.Marshal(items)
	result := CrushJSONArray(string(data), DefaultCrusherConfig())

	if result.WasModified {
		t.Fatal("non-object array should not be crushed")
	}
}

func TestCrushJSONArray_TopN(t *testing.T) {
	// Create array with score field
	items := make([]map[string]any, 50)
	for i := 0; i < 50; i++ {
		score := 0.1 + float64(i)*0.015
		if i == 5 || i == 15 {
			score = 0.95 // high relevance
		}
		items[i] = map[string]any{
			"id":         float64(i),
			"score":      score,
			"item_name":  "tool-result",
			"item_value": float64(i * 100),
		}
	}
	data, _ := json.Marshal(items)
	cfg := DefaultCrusherConfig()
	cfg.MaxItemsAfterCrush = 10

	result := CrushJSONArray(string(data), cfg)

	if !result.WasModified {
		t.Fatal("expected modification")
	}
	if result.ItemsAfter > 10 {
		t.Fatalf("expected <= 10 items, got %d", result.ItemsAfter)
	}
	t.Logf("top_n crushed %d -> %d items (ratio: %.2f)", result.ItemsBefore, result.ItemsAfter, result.SavingsRatio)
}

func TestCompressContent_JSONArray(t *testing.T) {
	items := make([]map[string]any, 50)
	for i := 0; i < 50; i++ {
		items[i] = map[string]any{"id": float64(i), "name": "test-item-" + fmt.Sprint(i), "value": float64(i * 2)}
	}
	data, _ := json.Marshal(items)

	compressed, modified, strategy := CompressContent(string(data), DefaultConfig())
	if !modified {
		t.Fatal("expected modification")
	}
	if !strings.HasPrefix(strategy, "json_") {
		t.Fatalf("unexpected strategy: %s", strategy)
	}

	// Verify valid JSON
	var parsed []any
	if err := json.Unmarshal([]byte(compressed), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(parsed) >= len(items) {
		t.Fatal("should have fewer items")
	}
	t.Logf("CompressContent %d -> %d items using %s", len(items), len(parsed), strategy)
}

func TestCompressContent_Code(t *testing.T) {
	content := `// This is a comment
func main() {
	// This should be removed
	fmt.Println("hello")
}

// Another comment
func helper() {
	return 42
}
`
	compressed, modified, strategy := CompressContent(content, DefaultConfig())
	t.Logf("Code compress: modified=%v strategy=%s compressed=%q", modified, strategy, compressed)
	if !modified {
		t.Fatal("expected modification")
	}
	if strings.Contains(compressed, "// This is a comment") {
		t.Fatal("comments should be removed")
	}
}

func TestCompressContent_SearchResults(t *testing.T) {
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, "src/file.go:10:func item() {")
	}
	content := strings.Join(lines, "\n")

	compressed, modified, _ := CompressContent(content, DefaultConfig())
	if !modified {
		t.Fatal("expected modification")
	}
	outLines := strings.Split(strings.TrimSpace(compressed), "\n")
	if len(outLines) > len(lines) {
		t.Fatal("should have fewer or equal lines")
	}
}

func TestCompressMessages(t *testing.T) {
	messages := []map[string]any{
		{"role": "system", "content": "You are a helpful assistant."},
		{"role": "user", "content": "Hi"},
		{"role": "tool", "content": `[{"id": 1, "name": "a", "value": 100}, {"id": 2, "name": "b", "value": 200}, {"id": 3, "name": "c", "value": 300}, {"id": 4, "name": "d", "value": 400}, {"id": 5, "name": "e", "value": 500}, {"id": 6, "name": "f", "value": 600}, {"id": 7, "name": "g", "value": 700}, {"id": 8, "name": "h", "value": 800}, {"id": 9, "name": "i", "value": 900}, {"id": 10, "name": "j", "value": 1000}, {"id": 11, "name": "k", "value": 1100}, {"id": 12, "name": "l", "value": 1200}, {"id": 13, "name": "m", "value": 1300}, {"id": 14, "name": "n", "value": 1400}, {"id": 15, "name": "o", "value": 1500}, {"id": 16, "name": "p", "value": 1600}]`},
		{"role": "assistant", "content": "Done"},
	}

	cfg := DefaultConfig()
	cfg.MinContentLength = 10
	cfg.Crusher.MinTokensToCrush = 10
	result := CompressMessages(messages, cfg)

	if result[0]["content"] != messages[0]["content"] {
		t.Fatal("system message should not change")
	}

	toolContent := result[2]["content"].(string)
	var parsed []any
	if err := json.Unmarshal([]byte(toolContent), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(parsed) >= 16 {
		t.Fatalf("expected fewer items, got %d", len(parsed))
	}
	t.Logf("messages crushed %d -> %d items", 16, len(parsed))
}

func TestCompressFieldNames(t *testing.T) {
	objects := []map[string]any{
		{"very_long_field_err": "a", "short": "b"},
		{"very_long_field_err": "c", "short": "d"},
	}

	result := compressFieldNames(objects)
	if len(result) != 2 {
		t.Fatal("should preserve item count")
	}

	// Long field should be shortened
	for _, obj := range result {
		for k := range obj {
			if strings.Contains(k, "very_long_field_err") {
				t.Fatalf("long field name should be shortened, got: %s", k)
			}
		}
	}
	t.Logf("long field shortened: %q", func() []string {
		var keys []string
		for k := range result[0] {
			keys = append(keys, k)
		}
		return keys
	}())
}

func TestCompressText(t *testing.T) {
	content := "Hello, this is just some text.\n\n\n\n\nMore text after blanks."
	compressed, modified, strategy := CompressContent(content, DefaultConfig())
	if compressed != content {
		t.Fatalf("plain text should not be modified for short content, modified=%v strategy=%s", modified, strategy)
	}
	_ = compressed
}
