// Package graph extracts structure from source code using AST parsing,
// builds a knowledge graph, runs community detection, and generates
// interactive visualizations.
//
// Reference: graphify (MIT License)
package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Node represents an entity in the knowledge graph.
type Node struct {
	ID       string            `json:"id"`
	Label    string            `json:"label"`
	Type     string            `json:"type"` // "function", "class", "module", "variable", "import", "concept"
	Language string            `json:"language,omitempty"`
	File     string            `json:"file,omitempty"`
	Line     int               `json:"line,omitempty"`
	Meta     map[string]string `json:"meta,omitempty"`
}

// Edge represents a relationship between nodes.
type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // "calls", "imports", "extends", "implements", "contains", "references"
	Weight int    `json:"weight,omitempty"`
}

// Graph is the complete knowledge graph.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Community represents a cluster of related nodes.
type Community struct {
	ID    int      `json:"id"`
	Name  string   `json:"name"`
	Nodes []string `json:"nodes"`
}

// Result holds the complete analysis output.
type Result struct {
	Graph       Graph       `json:"graph"`
	Communities []Community `json:"communities"`
	Stats       Stats       `json:"stats"`
}

// Stats holds graph statistics.
type Stats struct {
	TotalNodes    int `json:"total_nodes"`
	TotalEdges    int `json:"total_edges"`
	TotalFiles    int `json:"total_files"`
	Languages     int `json:"languages"`
	Communities   int `json:"communities"`
	AvgDegree     float64 `json:"avg_degree"`
	MaxDegree     int     `json:"max_degree"`
	GodNodes      []string `json:"god_nodes,omitempty"` // nodes with highest degree
}

// Analyzer configures the graph extraction.
type Analyzer struct {
	RootDir     string
	MaxDepth    int
	IncludeTests bool
	Languages   []string // empty = all supported
}

// NewAnalyzer creates an analyzer for the given directory.
func NewAnalyzer(rootDir string) *Analyzer {
	return &Analyzer{
		RootDir:     rootDir,
		MaxDepth:    10,
		IncludeTests: false,
	}
}

// Analyze runs the full extraction pipeline.
func (a *Analyzer) Analyze() (*Result, error) {
	graph := Graph{}

	// Walk the directory and extract from each file
	err := filepath.Walk(a.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			depth := strings.Count(path, string(os.PathSeparator)) - strings.Count(a.RootDir, string(os.PathSeparator))
			if depth > a.MaxDepth {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip test files if configured
		if !a.IncludeTests && (strings.HasSuffix(path, "_test.go") ||
			strings.HasSuffix(path, ".test.js") ||
			strings.HasSuffix(path, ".test.ts") ||
			strings.HasSuffix(path, "test_") ||
			strings.Contains(path, "/test/") ||
			strings.Contains(path, "/tests/")) {
			return nil
		}

		// Extract from file
		nodes, edges := extractFromFile(path, a.RootDir)
		graph.Nodes = append(graph.Nodes, nodes...)
		graph.Edges = append(graph.Edges, edges...)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk failed: %w", err)
	}

	// Deduplicate nodes
	graph = deduplicate(graph)

	// Run community detection
	communities := detectCommunities(&graph)

	// Compute stats
	stats := computeStats(&graph, communities)

	return &Result{
		Graph:       graph,
		Communities: communities,
		Stats:       stats,
	}, nil
}

// extractFromFile extracts nodes and edges from a single file.
func extractFromFile(path, rootDir string) ([]Node, []Edge) {
	ext := strings.ToLower(filepath.Ext(path))
	relPath, _ := filepath.Rel(rootDir, path)

	switch ext {
	case ".go":
		return extractGo(path, relPath)
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		return extractJS(path, relPath)
	case ".py":
		return extractPython(path, relPath)
	case ".rs":
		return extractRust(path, relPath)
	case ".java":
		return extractJava(path, relPath)
	default:
		return nil, nil
	}
}

// extractGo extracts Go source file structure.
func extractGo(path, relPath string) ([]Node, []Edge) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}

	text := string(content)
	lines := strings.Split(text, "\n")
	nodes := []Node{}
	edges := []Edge{}

	moduleName := strings.TrimSuffix(relPath, filepath.Ext(relPath))
	moduleName = strings.ReplaceAll(moduleName, "/", ".")

	// Add module node
	nodes = append(nodes, Node{
		ID:       "module:" + moduleName,
		Label:    moduleName,
		Type:     "module",
		Language: "go",
		File:     relPath,
	})

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Function declarations
		if strings.HasPrefix(trimmed, "func ") {
			name := extractFunctionName(trimmed)
			if name != "" {
				nodeID := "func:" + moduleName + "." + name
				nodes = append(nodes, Node{
					ID:       nodeID,
					Label:    name,
					Type:     "function",
					Language: "go",
					File:     relPath,
					Line:     i + 1,
				})
				edges = append(edges, Edge{
					Source: "module:" + moduleName,
					Target: nodeID,
					Type:   "contains",
				})
			}
		}

		// Type declarations (structs, interfaces)
		if strings.HasPrefix(trimmed, "type ") {
			name := extractTypeName(trimmed)
			if name != "" {
				nodeID := "type:" + moduleName + "." + name
				nodes = append(nodes, Node{
					ID:       nodeID,
					Label:    name,
					Type:     "class",
					Language: "go",
					File:     relPath,
					Line:     i + 1,
				})
				edges = append(edges, Edge{
					Source: "module:" + moduleName,
					Target: nodeID,
					Type:   "contains",
				})
			}
		}

		// Import statements
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "\"") {
			pkg := extractImportPath(trimmed)
			if pkg != "" {
				nodes = append(nodes, Node{
					ID:       "import:" + pkg,
					Label:    pkg,
					Type:     "import",
					Language: "go",
					File:     relPath,
				})
				edges = append(edges, Edge{
					Source: "module:" + moduleName,
					Target: "import:" + pkg,
					Type:   "imports",
				})
			}
		}
	}

	return nodes, edges
}

// extractJS extracts JavaScript/TypeScript source file structure.
func extractJS(path, relPath string) ([]Node, []Edge) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}

	text := string(content)
	lines := strings.Split(text, "\n")
	nodes := []Node{}
	edges := []Edge{}

	moduleName := strings.TrimSuffix(relPath, filepath.Ext(relPath))
	moduleName = strings.ReplaceAll(moduleName, "/", ".")

	// Add module node
	nodes = append(nodes, Node{
		ID:       "module:" + moduleName,
		Label:    moduleName,
		Type:     "module",
		Language: "javascript",
		File:     relPath,
	})

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Function declarations
		if strings.HasPrefix(trimmed, "function ") || strings.HasPrefix(trimmed, "async function ") {
			name := extractJSFunctionName(trimmed)
			if name != "" {
				nodeID := "func:" + moduleName + "." + name
				nodes = append(nodes, Node{
					ID:       nodeID,
					Label:    name,
					Type:     "function",
					Language: "javascript",
					File:     relPath,
					Line:     i + 1,
				})
				edges = append(edges, Edge{
					Source: "module:" + moduleName,
					Target: nodeID,
					Type:   "contains",
				})
			}
		}

		// Class declarations
		if strings.HasPrefix(trimmed, "class ") {
			name := extractClassName(trimmed)
			if name != "" {
				nodeID := "class:" + moduleName + "." + name
				nodes = append(nodes, Node{
					ID:       nodeID,
					Label:    name,
					Type:     "class",
					Language: "javascript",
					File:     relPath,
					Line:     i + 1,
				})
				edges = append(edges, Edge{
					Source: "module:" + moduleName,
					Target: nodeID,
					Type:   "contains",
				})
			}
		}

		// Import statements
		if strings.HasPrefix(trimmed, "import ") {
			pkg := extractJSImportPath(trimmed)
			if pkg != "" {
				nodes = append(nodes, Node{
					ID:       "import:" + pkg,
					Label:    pkg,
					Type:     "import",
					Language: "javascript",
					File:     relPath,
				})
				edges = append(edges, Edge{
					Source: "module:" + moduleName,
					Target: "import:" + pkg,
					Type:   "imports",
				})
			}
		}
	}

	return nodes, edges
}

// extractPython extracts Python source file structure.
func extractPython(path, relPath string) ([]Node, []Edge) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}

	text := string(content)
	lines := strings.Split(text, "\n")
	nodes := []Node{}
	edges := []Edge{}

	moduleName := strings.TrimSuffix(relPath, filepath.Ext(relPath))
	moduleName = strings.ReplaceAll(moduleName, "/", ".")

	// Add module node
	nodes = append(nodes, Node{
		ID:       "module:" + moduleName,
		Label:    moduleName,
		Type:     "module",
		Language: "python",
		File:     relPath,
	})

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Function definitions
		if strings.HasPrefix(trimmed, "def ") {
			name := extractPythonDefName(trimmed)
			if name != "" {
				nodeID := "func:" + moduleName + "." + name
				nodes = append(nodes, Node{
					ID:       nodeID,
					Label:    name,
					Type:     "function",
					Language: "python",
					File:     relPath,
					Line:     i + 1,
				})
				edges = append(edges, Edge{
					Source: "module:" + moduleName,
					Target: nodeID,
					Type:   "contains",
				})
			}
		}

		// Class definitions
		if strings.HasPrefix(trimmed, "class ") {
			name := extractPythonClassName(trimmed)
			if name != "" {
				nodeID := "class:" + moduleName + "." + name
				nodes = append(nodes, Node{
					ID:       nodeID,
					Label:    name,
					Type:     "class",
					Language: "python",
					File:     relPath,
					Line:     i + 1,
				})
				edges = append(edges, Edge{
					Source: "module:" + moduleName,
					Target: nodeID,
					Type:   "contains",
				})
			}
		}

		// Import statements
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ") {
			pkg := extractPythonImportPath(trimmed)
			if pkg != "" {
				nodes = append(nodes, Node{
					ID:       "import:" + pkg,
					Label:    pkg,
					Type:     "import",
					Language: "python",
					File:     relPath,
				})
				edges = append(edges, Edge{
					Source: "module:" + moduleName,
					Target: "import:" + pkg,
					Type:   "imports",
				})
			}
		}
	}

	return nodes, edges
}

// extractRust extracts Rust source file structure.
func extractRust(path, relPath string) ([]Node, []Edge) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}

	text := string(content)
	lines := strings.Split(text, "\n")
	nodes := []Node{}
	edges := []Edge{}

	moduleName := strings.TrimSuffix(relPath, filepath.Ext(relPath))
	moduleName = strings.ReplaceAll(moduleName, "/", ".")

	nodes = append(nodes, Node{
		ID:       "module:" + moduleName,
		Label:    moduleName,
		Type:     "module",
		Language: "rust",
		File:     relPath,
	})

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "fn ") || strings.HasPrefix(trimmed, "pub fn ") {
			name := extractRustFnName(trimmed)
			if name != "" {
				nodeID := "func:" + moduleName + "." + name
				nodes = append(nodes, Node{
					ID:       nodeID,
					Label:    name,
					Type:     "function",
					Language: "rust",
					File:     relPath,
					Line:     i + 1,
				})
				edges = append(edges, Edge{
					Source: "module:" + moduleName,
					Target: nodeID,
					Type:   "contains",
				})
			}
		}

		if strings.HasPrefix(trimmed, "struct ") || strings.HasPrefix(trimmed, "pub struct ") {
			name := extractRustStructName(trimmed)
			if name != "" {
				nodeID := "struct:" + moduleName + "." + name
				nodes = append(nodes, Node{
					ID:       nodeID,
					Label:    name,
					Type:     "class",
					Language: "rust",
					File:     relPath,
					Line:     i + 1,
				})
				edges = append(edges, Edge{
					Source: "module:" + moduleName,
					Target: nodeID,
					Type:   "contains",
				})
			}
		}
	}

	return nodes, edges
}

// extractJava extracts Java source file structure.
func extractJava(path, relPath string) ([]Node, []Edge) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}

	text := string(content)
	lines := strings.Split(text, "\n")
	nodes := []Node{}
	edges := []Edge{}

	moduleName := strings.TrimSuffix(relPath, filepath.Ext(relPath))
	moduleName = strings.ReplaceAll(moduleName, "/", ".")

	nodes = append(nodes, Node{
		ID:       "module:" + moduleName,
		Label:    moduleName,
		Type:     "module",
		Language: "java",
		File:     relPath,
	})

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "public ") || strings.HasPrefix(trimmed, "private ") || strings.HasPrefix(trimmed, "protected ") {
			if strings.Contains(trimmed, "void ") || strings.Contains(trimmed, "static ") {
				name := extractJavaMethodName(trimmed)
				if name != "" {
					nodeID := "func:" + moduleName + "." + name
					nodes = append(nodes, Node{
						ID:       nodeID,
						Label:    name,
						Type:     "function",
						Language: "java",
						File:     relPath,
						Line:     i + 1,
					})
					edges = append(edges, Edge{
						Source: "module:" + moduleName,
						Target: nodeID,
						Type:   "contains",
					})
				}
			}
		}
	}

	return nodes, edges
}

// --- Name extraction helpers ---

func extractFunctionName(line string) string {
	line = strings.TrimPrefix(line, "func ")
	line = strings.TrimPrefix(line, "func (")
	if idx := strings.Index(line, ")"); idx != -1 {
		line = line[idx+1:]
	}
	line = strings.TrimSpace(line)
	if idx := strings.IndexAny(line, "( {"); idx != -1 {
		return line[:idx]
	}
	return line
}

func extractTypeName(line string) string {
	line = strings.TrimPrefix(line, "type ")
	line = strings.TrimSpace(line)
	if idx := strings.IndexAny(line, " {"); idx != -1 {
		return line[:idx]
	}
	return line
}

func extractImportPath(line string) string {
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "\"")
	if idx := strings.Index(line, "/"); idx != -1 {
		parts := strings.Split(line, "/")
		if len(parts) >= 3 {
			return strings.Join(parts[:3], "/")
		}
	}
	return line
}

func extractJSFunctionName(line string) string {
	line = strings.TrimPrefix(line, "async ")
	line = strings.TrimPrefix(line, "function ")
	if idx := strings.IndexAny(line, "( {"); idx != -1 {
		return line[:idx]
	}
	return line
}

func extractClassName(line string) string {
	line = strings.TrimPrefix(line, "class ")
	if idx := strings.IndexAny(line, " {extends"); idx != -1 {
		return line[:idx]
	}
	return line
}

func extractJSImportPath(line string) string {
	if idx := strings.Index(line, "from "); idx != -1 {
		line = line[idx+5:]
	}
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "\"'")
	return line
}

func extractPythonDefName(line string) string {
	line = strings.TrimPrefix(line, "async ")
	line = strings.TrimPrefix(line, "def ")
	if idx := strings.IndexAny(line, "( :"); idx != -1 {
		return line[:idx]
	}
	return line
}

func extractPythonClassName(line string) string {
	line = strings.TrimPrefix(line, "class ")
	if idx := strings.IndexAny(line, "( :"); idx != -1 {
		return line[:idx]
	}
	return line
}

func extractPythonImportPath(line string) string {
	if strings.HasPrefix(line, "from ") {
		line = strings.TrimPrefix(line, "from ")
		if idx := strings.Index(line, " "); idx != -1 {
			return line[:idx]
		}
	}
	line = strings.TrimPrefix(line, "import ")
	if idx := strings.Index(line, " "); idx != -1 {
		return line[:idx]
	}
	return line
}

func extractRustFnName(line string) string {
	line = strings.TrimPrefix(line, "pub ")
	line = strings.TrimPrefix(line, "fn ")
	if idx := strings.IndexAny(line, "( <"); idx != -1 {
		return line[:idx]
	}
	return line
}

func extractRustStructName(line string) string {
	line = strings.TrimPrefix(line, "pub ")
	line = strings.TrimPrefix(line, "struct ")
	if idx := strings.IndexAny(line, " {<"); idx != -1 {
		return line[:idx]
	}
	return line
}

func extractJavaMethodName(line string) string {
	// Remove modifiers
	for _, mod := range []string{"public ", "private ", "protected ", "static ", "final ", "abstract "} {
		line = strings.TrimPrefix(line, mod)
	}
	if idx := strings.Index(line, "("); idx != -1 {
		parts := strings.Fields(line[:idx])
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return ""
}

// --- Graph operations ---

func deduplicate(g Graph) Graph {
	seen := make(map[string]bool)
	var nodes []Node
	for _, n := range g.Nodes {
		if !seen[n.ID] {
			seen[n.ID] = true
			nodes = append(nodes, n)
		}
	}
	g.Nodes = nodes
	return g
}

// detectCommunities finds clusters of related nodes using a simple
// label propagation algorithm (Leiden-lite).
func detectCommunities(g *Graph) []Community {
	if len(g.Nodes) == 0 {
		return nil
	}

	// Build adjacency list
	adj := make(map[string][]string)
	for _, e := range g.Edges {
		adj[e.Source] = append(adj[e.Source], e.Target)
		adj[e.Target] = append(adj[e.Target], e.Source)
	}

	// Initialize each node as its own community
	community := make(map[string]int)
	for i, n := range g.Nodes {
		community[n.ID] = i
	}

	// Simple label propagation (10 iterations)
	for iter := 0; iter < 10; iter++ {
		changed := false
		for _, n := range g.Nodes {
			neighbors := adj[n.ID]
			if len(neighbors) == 0 {
				continue
			}

			// Count neighbor communities
			counts := make(map[int]int)
			for _, nb := range neighbors {
				counts[community[nb]]++
			}

			// Find most common community
			best := community[n.ID]
			bestCount := 0
			for c, cnt := range counts {
				if cnt > bestCount {
					best = c
					bestCount = cnt
				}
			}

			if best != community[n.ID] {
				community[n.ID] = best
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	// Group nodes by community
	commMap := make(map[int][]string)
	for _, n := range g.Nodes {
		c := community[n.ID]
		commMap[c] = append(commMap[c], n.ID)
	}

	var communities []Community
	for c, nodeIDs := range commMap {
		name := generateCommunityName(nodeIDs, g)
		communities = append(communities, Community{
			ID:    c,
			Name:  name,
			Nodes: nodeIDs,
		})
	}

	return communities
}

func generateCommunityName(nodeIDs []string, g *Graph) string {
	// Use the most connected node's label as the community name
	if len(nodeIDs) == 0 {
		return "Unknown"
	}

	// Find node with highest degree
	best := nodeIDs[0]
	bestDegree := 0
	degree := make(map[string]int)
	for _, e := range g.Edges {
		for _, id := range nodeIDs {
			if e.Source == id || e.Target == id {
				degree[id]++
			}
		}
	}
	for _, id := range nodeIDs {
		if degree[id] > bestDegree {
			best = id
			bestDegree = degree[id]
		}
	}

	// Find the node label
	for _, n := range g.Nodes {
		if n.ID == best {
			return n.Label
		}
	}
	return best
}

func computeStats(g *Graph, communities []Community) Stats {
	s := Stats{
		TotalNodes:  len(g.Nodes),
		TotalEdges:  len(g.Edges),
		Communities: len(communities),
	}

	// Count languages and files
	langs := make(map[string]bool)
	files := make(map[string]bool)
	for _, n := range g.Nodes {
		if n.Language != "" {
			langs[n.Language] = true
		}
		if n.File != "" {
			files[n.File] = true
		}
	}
	s.Languages = len(langs)
	s.TotalFiles = len(files)

	// Compute degree
	degree := make(map[string]int)
	for _, e := range g.Edges {
		degree[e.Source]++
		degree[e.Target]++
	}

	totalDegree := 0
	for _, d := range degree {
		totalDegree += d
		if d > s.MaxDegree {
			s.MaxDegree = d
		}
	}
	if len(g.Nodes) > 0 {
		s.AvgDegree = float64(totalDegree) / float64(len(g.Nodes))
	}

	// Find god nodes (top 5 by degree)
	type nodeDegree struct {
		ID     string
		Degree int
	}
	var nodelist []nodeDegree
	for id, d := range degree {
		nodelist = append(nodelist, nodeDegree{id, d})
	}
	sort.Slice(nodelist, func(i, j int) bool {
		return nodelist[i].Degree > nodelist[j].Degree
	})
	for i := 0; i < len(nodelist) && i < 5; i++ {
		// Find node label
		for _, n := range g.Nodes {
			if n.ID == nodelist[i].ID {
				s.GodNodes = append(s.GodNodes, n.Label)
				break
			}
		}
	}

	return s
}

// --- Output ---

// SaveJSON saves the result as JSON.
func (r *Result) SaveJSON(path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// SaveHTML generates an interactive HTML visualization.
func (r *Result) SaveHTML(path string) error {
	html := r.GenerateHTML()
	return os.WriteFile(path, []byte(html), 0644)
}

// GenerateHTML returns an interactive HTML visualization as a string.
func (r *Result) GenerateHTML() string {
	data, err := json.Marshal(r.Graph)
	if err != nil {
		return "<html><body><p>Error generating visualization</p></body></html>"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>Knowledge Graph</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#0f1117;color:#e5e5e5;font-family:system-ui,sans-serif}
canvas{display:block}
.info{position:fixed;top:1rem;left:1rem;background:#1a1d27;padding:1rem;border-radius:8px;font-size:.85rem;max-width:300px}
.info h2{font-size:1rem;margin-bottom:.5rem}
.stat{color:#9ca3af;font-size:.75rem}
.legend{position:fixed;bottom:1rem;left:1rem;background:#1a1d27;padding:.75rem;border-radius:8px;font-size:.75rem}
.legend-item{display:flex;align-items:center;gap:.5rem;margin:.25rem 0}
.legend-dot{width:10px;height:10px;border-radius:50%%}
</style>
</head>
<body>
<div class="info">
<h2>Knowledge Graph</h2>
<p class="stat">%d nodes, %d edges</p>
<p class="stat">%d communities, %d languages</p>
</div>
<canvas id="graph"></canvas>
<script>
const graph = %s;
const canvas = document.getElementById("graph");
const ctx = canvas.getContext("2d");
canvas.width = window.innerWidth;
canvas.height = window.innerHeight;

const colors = {module:"#3b82f6",function:"#22c55e",class:"#eab308",import:"#a855f7",variable:"#f97316"};
const nodes = graph.nodes.map((n,i)=>({...n,x:Math.random()*canvas.width,y:Math.random()*canvas.height,vx:0,vy:0,color:colors[n.type]||"#6b7280"}));
const nodeMap = {};
nodes.forEach((n,i)=>nodeMap[n.id]=i);
const edges = graph.edges.filter(e=>nodeMap[e.source]!==undefined&&nodeMap[e.target]!==undefined);

function simulate(){
for(let i=0;i<50;i++){
nodes.forEach(n=>{n.vx*=0.9;n.vy*=0.9});
edges.forEach(e=>{
const a=nodes[nodeMap[e.source]],b=nodes[nodeMap[e.target]];
const dx=b.x-a.x,dy=b.y-a.y;
const d=Math.sqrt(dx*dx+dy*dy)||1;
const f=(d-100)*0.01;
a.vx+=dx/d*f;a.vy+=dy/d*f;
b.vx-=dx/d*f;b.vy-=dy/d*f;
});
nodes.forEach(n=>{
nodes.forEach(m=>{
if(n===m)return;
const dx=m.x-n.x,dy=m.y-n.y;
const d=Math.sqrt(dx*dx+dy*dy)||1;
if(d<80){const f=0.5/d;n.vx-=dx*f;n.vy-=dy*f;}
});
});
nodes.forEach(n=>{n.x+=n.vx;n.y+=n.vy;n.x=Math.max(20,Math.min(canvas.width-20,n.x));n.y=Math.max(20,Math.min(canvas.height-20,n.y));});
}
}

function draw(){
ctx.clearRect(0,0,canvas.width,canvas.height);
ctx.strokeStyle="#2a2d3a";ctx.lineWidth=0.5;
edges.forEach(e=>{
const a=nodes[nodeMap[e.source]],b=nodes[nodeMap[e.target]];
ctx.beginPath();ctx.moveTo(a.x,a.y);ctx.lineTo(b.x,b.y);ctx.stroke();
});
nodes.forEach(n=>{
ctx.beginPath();ctx.arc(n.x,n.y,6,0,Math.PI*2);ctx.fillStyle=n.color;ctx.fill();
ctx.fillStyle="#e5e5e5";ctx.font="9px system-ui";ctx.textAlign="center";ctx.fillText(n.label,n.x,n.y+16);
});
}

simulate();draw();
</script>
</body>
</html>`,
		r.Stats.TotalNodes, r.Stats.TotalEdges, r.Stats.Communities, r.Stats.Languages, string(data))
}
