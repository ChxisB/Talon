// Package viz generates self-contained HTML diagrams from JSON intermediate
// representations. Supports architecture, workflow, sequence, dataflow, and
// lifecycle diagram types.
//
// Reference: archify (MIT License)
package viz

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DiagramType enumerates the supported diagram types.
type DiagramType string

const (
	TypeArchitecture DiagramType = "architecture"
	TypeWorkflow     DiagramType = "workflow"
	TypeSequence     DiagramType = "sequence"
	TypeDataflow     DiagramType = "dataflow"
	TypeLifecycle    DiagramType = "lifecycle"
)

// Meta holds diagram metadata.
type Meta struct {
	Title    string  `json:"title"`
	Subtitle string  `json:"subtitle,omitempty"`
	Output   string  `json:"output,omitempty"`
	ViewBox  *[2]int `json:"viewBox,omitempty"`
}

// Component represents a diagram node (architecture type).
type Component struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`
	Label    string      `json:"label"`
	Sublabel string      `json:"sublabel,omitempty"`
	Tag      string      `json:"tag,omitempty"`
	Pos      [2]float64  `json:"pos"`
	Size     *[2]float64 `json:"size,omitempty"`
}

// Boundary wraps a group of components.
type Boundary struct {
	Kind  string   `json:"kind"`
	Label string   `json:"label"`
	Wraps []string `json:"wraps"`
	Pad   *float64 `json:"pad,omitempty"`
}

// Connection links two components.
type Connection struct {
	From         string        `json:"from"`
	To           string        `json:"to"`
	Label        string        `json:"label,omitempty"`
	Variant      string        `json:"variant,omitempty"`
	FromSide     string        `json:"fromSide,omitempty"`
	ToSide       string        `json:"toSide,omitempty"`
	Route        string        `json:"route,omitempty"`
	Via          *[][2]float64 `json:"via,omitempty"`
	LabelAt      *Point        `json:"labelAt,omitempty"`
	LabelDx      float64       `json:"labelDx,omitempty"`
	LabelDy      float64       `json:"labelDy,omitempty"`
	LabelSegment *int          `json:"labelSegment,omitempty"`
	Width        float64       `json:"width,omitempty"`
}

// Point is a 2D coordinate.
type Point = [2]float64

// Card is an info card rendered below the diagram.
type Card struct {
	Dot   string   `json:"dot"`
	Title string   `json:"title"`
	Items []string `json:"items"`
}

// Step is a node in workflow/sequence/lifecycle diagrams.
type Step struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Type    string `json:"type,omitempty"`
	Shape   string `json:"shape,omitempty"`
	Subtext string `json:"subtext,omitempty"`
	Pos     Point  `json:"pos"`
}

// Transition is an edge in workflow/lifecycle diagrams.
type Transition struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label,omitempty"`
}

// SequenceMessage is a message in a sequence diagram.
type SequenceMessage struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"`
	Type  string `json:"type,omitempty"` // "sync", "async", "return"
}

// SequenceParticipant is a participant in a sequence diagram.
type SequenceParticipant struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Pos   Point  `json:"pos"`
	Type  string `json:"type,omitempty"` // "actor", "system", "database"
}

// FlowNode is a node in a dataflow diagram.
type FlowNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type,omitempty"` // "source", "process", "store", "sink"
	Pos   Point  `json:"pos"`
}

// FlowEdge is an edge in a dataflow diagram.
type FlowEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label,omitempty"`
}

// Diagram is the top-level IR for all diagram types.
type Diagram struct {
	SchemaVersion int                   `json:"schema_version"`
	DiagramType   DiagramType           `json:"diagram_type"`
	Meta          Meta                  `json:"meta"`
	Components    []Component           `json:"components,omitempty"`
	Boundaries    []Boundary            `json:"boundaries,omitempty"`
	Connections   []Connection          `json:"connections,omitempty"`
	Steps         []Step                `json:"steps,omitempty"`
	Transitions   []Transition          `json:"transitions,omitempty"`
	Messages      []SequenceMessage     `json:"messages,omitempty"`
	Participants  []SequenceParticipant `json:"participants,omitempty"`
	Nodes         []FlowNode            `json:"nodes,omitempty"`
	Edges         []FlowEdge            `json:"edges,omitempty"`
	Cards         []Card                `json:"cards,omitempty"`
}

// Generate creates a self-contained HTML file from a Diagram IR.
func Generate(diagram *Diagram) ([]byte, error) {
	if diagram.Meta.Title == "" {
		return nil, fmt.Errorf("diagram must have a meta.title")
	}

	var svg string
	switch diagram.DiagramType {
	case TypeArchitecture:
		svg = renderArchitecture(diagram)
	case TypeWorkflow:
		svg = renderWorkflow(diagram)
	case TypeSequence:
		svg = renderSequence(diagram)
	case TypeDataflow:
		svg = renderDataflow(diagram)
	case TypeLifecycle:
		svg = renderLifecycle(diagram)
	default:
		return nil, fmt.Errorf("unsupported diagram type: %s", diagram.DiagramType)
	}

	html := applyTemplate(diagram.Meta, svg, renderCards(diagram.Cards), string(diagram.DiagramType))
	return []byte(html), nil
}

// Parse parses a JSON IR string into a Diagram.
func Parse(data []byte) (*Diagram, error) {
	var d Diagram
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("invalid diagram JSON: %w", err)
	}
	return &d, nil
}

// --- Geometry helpers ---

func asArray[T any](v []T) []T {
	if v == nil {
		return []T{}
	}
	return v
}

func esc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

func anchor(x, y, w, h float64, side string) Point {
	cx, cy := x+w/2, y+h/2
	switch side {
	case "left":
		return Point{x, cy}
	case "right":
		return Point{x + w, cy}
	case "top":
		return Point{cx, y}
	case "bottom":
		return Point{cx, y + h}
	default:
		return Point{x + w, cy}
	}
}

func defaultFromSide(fromX, fromY, fromW, fromH, toX, toY float64) string {
	fromCx := fromX + fromW/2
	toCx := toX
	if toCx < fromCx {
		return "left"
	}
	if toCx > fromCx {
		return "right"
	}
	if toY > fromY+fromH/2 {
		return "bottom"
	}
	return "top"
}

func defaultToSide(fromX, fromY, fromW, fromH, toX, toY, toW, toH float64) string {
	fromCx := fromX + fromW/2
	toCx := toX + toW/2
	if toCx < fromCx {
		return "right"
	}
	if toCx > fromCx {
		return "left"
	}
	if toY > fromY+fromH/2 {
		return "top"
	}
	return "bottom"
}

func polylinePath(points []Point) string {
	var b strings.Builder
	for i, p := range points {
		if i == 0 {
			b.WriteString(fmt.Sprintf("M %g %g", p[0], p[1]))
		} else {
			b.WriteString(fmt.Sprintf(" L %g %g", p[0], p[1]))
		}
	}
	return b.String()
}

func roundedPath(points []Point, radius float64) string {
	if len(points) < 3 || radius <= 0 {
		return polylinePath(points)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("M %g %g", points[0][0], points[0][1]))

	for i := 1; i < len(points)-1; i++ {
		px, py := points[i-1][0], points[i-1][1]
		cx, cy := points[i][0], points[i][1]
		nx, ny := points[i+1][0], points[i+1][1]

		prevLen := dist(px, py, cx, cy)
		nextLen := dist(cx, cy, nx, ny)
		r := radius
		if prevLen/2 < r {
			r = prevLen / 2
		}
		if nextLen/2 < r {
			r = nextLen / 2
		}

		if r < 1 {
			b.WriteString(fmt.Sprintf(" L %g %g", cx, cy))
			continue
		}

		bx := cx - (cx-px)/prevLen*r
		by := cy - (cy-py)/prevLen*r
		ax := cx + (nx-cx)/nextLen*r
		ay := cy + (ny-cy)/nextLen*r

		b.WriteString(fmt.Sprintf(" L %g %g", bx, by))
		b.WriteString(fmt.Sprintf(" Q %g %g %g %g", cx, cy, ax, ay))
	}

	end := points[len(points)-1]
	b.WriteString(fmt.Sprintf(" L %g %g", end[0], end[1]))
	return b.String()
}

func dist(x1, y1, x2, y2 float64) float64 {
	dx, dy := x2-x1, y2-y1
	return dx*dx + dy*dy
}

func labelPoint(conn *Connection, points []Point) Point {
	if conn.LabelAt != nil {
		return *conn.LabelAt
	}
	if len(points) == 2 {
		return Point{
			(points[0][0]+points[1][0])/2 + conn.LabelDx,
			points[0][1] - 10 + conn.LabelDy,
		}
	}
	seg := 1
	if conn.LabelSegment != nil {
		seg = *conn.LabelSegment
	}
	if seg >= len(points)-1 {
		seg = len(points) - 2
	}
	if seg < 0 {
		seg = 0
	}
	a, b := points[seg], points[seg+1]
	return Point{
		(a[0]+b[0])/2 + conn.LabelDx,
		(a[1]+b[1])/2 - 10 + conn.LabelDy,
	}
}

// --- Architecture renderer ---

func renderArchitecture(d *Diagram) string {
	const (
		defaultW    = 120
		defaultH    = 60
		margin      = 40
		boundaryPad = 30
		extraBottom = 20
		legendH     = 28
	)

	type measuredComponent struct {
		Component
		x, y, width, height, cx, cy float64
	}

	compMap := make(map[string]*measuredComponent)
	for _, c := range asArray(d.Components) {
		w, h := 120.0, 60.0
		if c.Size != nil {
			w, h = c.Size[0], c.Size[1]
		}
		mc := &measuredComponent{
			Component: c,
			x:         c.Pos[0], y: c.Pos[1], width: w, height: h,
			cx: c.Pos[0] + w/2, cy: c.Pos[1] + h/2,
		}
		compMap[c.ID] = mc
	}

	// Compute boundaries
	type boundaryRect struct {
		Boundary
		x, y, width, height float64
	}
	var bounds []boundaryRect
	for _, b := range asArray(d.Boundaries) {
		members := make([]*measuredComponent, 0)
		for _, id := range asArray(b.Wraps) {
			if mc, ok := compMap[id]; ok {
				members = append(members, mc)
			}
		}
		if len(members) == 0 {
			continue
		}
		minX, minY := members[0].x, members[0].y
		maxX, maxY := members[0].x+members[0].width, members[0].y+members[0].height
		for _, m := range members[1:] {
			if m.x < minX {
				minX = m.x
			}
			if m.y < minY {
				minY = m.y
			}
			if m.x+m.width > maxX {
				maxX = m.x + m.width
			}
			if m.y+m.height > maxY {
				maxY = m.y + m.height
			}
		}
		pad := 30.0
		bounds = append(bounds, boundaryRect{
			Boundary: b,
			x:        minX - pad, y: minY - pad,
			width: maxX - minX + pad*2, height: maxY - minY + pad + 20,
		})
	}

	// Auto viewBox
	maxX, maxY := 0.0, 0.0
	for _, mc := range compMap {
		if mc.x+mc.width > maxX {
			maxX = mc.x + mc.width
		}
		if mc.y+mc.height > maxY {
			maxY = mc.y + mc.height
		}
	}
	for _, b := range bounds {
		if b.x+b.width > maxX {
			maxX = b.x + b.width
		}
		if b.y+b.height > maxY {
			maxY = b.y + b.height
		}
	}
	vbW, vbH := maxX+margin, maxY+margin+legendH

	// Compute legend width before rendering so viewBox can accommodate it
	legendLabel := func(t string) string {
		m := map[string]string{
			"frontend": "Frontend", "backend": "Backend", "database": "Database",
			"cloud": "Cloud", "security": "Security", "messagebus": "Message Bus", "external": "External",
		}
		if l, ok := m[t]; ok {
			return l
		}
		return t
	}
	legendWidth := margin + 30.0 // "Legend" label + first gap
	used := make(map[string]bool)
	for _, mc := range compMap {
		if used[mc.Type] {
			continue
		}
		used[mc.Type] = true
		legendWidth += 30 + float64(len(legendLabel(mc.Type)))*5 + 28
	}
	if legendWidth > vbW {
		vbW = legendWidth
	}

	// Render SVG
	var svg strings.Builder
	svg.WriteString(fmt.Sprintf(`<svg viewBox="0 0 %g %g" role="img" aria-label="%s — architecture diagram">`, vbW, vbH, esc(d.Meta.Title)))
	svg.WriteString(renderDefs())
	svg.WriteString(`<rect width="100%" height="100%" fill="url(#grid)"/>`)

	// Boundaries
	for _, b := range bounds {
		cls := "c-region"
		if b.Kind == "security-group" {
			cls = "c-security-group"
		}
		rx := 12.0
		if b.Kind == "security-group" {
			rx = 8
		}
		svg.WriteString(fmt.Sprintf(`<rect x="%g" y="%g" width="%g" height="%g" rx="%g" class="%s" stroke-width="1"/>`,
			b.x, b.y, b.width, b.height, rx, cls))
		svg.WriteString(fmt.Sprintf(`<text x="%g" y="%g" class="t-cloud" font-size="9" font-weight="600">%s</text>`,
			b.x+8, b.y+18, esc(b.Label)))
	}

	// Connections
	connCache := make(map[string][]Point)
	for _, conn := range asArray(d.Connections) {
		from, ok1 := compMap[conn.From]
		to, ok2 := compMap[conn.To]
		if !ok1 || !ok2 {
			continue
		}
		fromSide := conn.FromSide
		if fromSide == "" || fromSide == "auto" {
			fromSide = defaultFromSide(from.x, from.y, from.width, from.height, to.x, to.y)
		}
		toSide := conn.ToSide
		if toSide == "" || toSide == "auto" {
			toSide = defaultToSide(from.x, from.y, from.width, from.height, to.x, to.y, to.width, to.height)
		}
		start := anchor(from.x, from.y, from.width, from.height, fromSide)
		end := anchor(to.x, to.y, to.width, to.height, toSide)

		points := []Point{start}
		if conn.Via != nil {
			for _, v := range *conn.Via {
				points = append(points, v)
			}
		} else {
			// Auto routing
			if abs(start[0]-end[0]) >= 4 && abs(start[1]-end[1]) >= 4 {
				midX := (start[0] + end[0]) / 2
				points = append(points, Point{midX, start[1]}, Point{midX, end[1]})
			}
		}
		points = append(points, end)

		cls, marker := "a-default", "arrowhead"
		if conn.Variant == "emphasis" {
			cls, marker = "a-emphasis", "arrowhead-emphasis"
		} else if conn.Variant == "security" {
			cls, marker = "a-security", "arrowhead-security"
		} else if conn.Variant == "dashed" {
			cls, marker = "a-dashed", "arrowhead-dashed"
		}
		sw := 1.5
		if conn.Width > 0 {
			sw = conn.Width
		} else if conn.Variant == "emphasis" {
			sw = 1.8
		}

		svg.WriteString(fmt.Sprintf(`<path d="%s" class="%s" stroke-width="%g" marker-end="url(#%s)"/>`,
			roundedPath(points, 8), cls, sw, marker))

		connKey := conn.From + "->" + conn.To
		connCache[connKey] = points

		// Connection label
		if conn.Label != "" {
			lp := labelPoint(&conn, points)
			w := 30.0
			if len(conn.Label)*6 > 30 {
				w = float64(len(conn.Label)*6 + 10)
			}
			svg.WriteString(fmt.Sprintf(`<rect x="%g" y="%g" width="%g" height="14" rx="3" class="c-mask"/>`,
				lp[0]-w/2, lp[1]-10, w))
			accent := "t-muted"
			if conn.Variant == "security" {
				accent = "t-security"
			} else if conn.Variant == "emphasis" {
				accent = "t-backend"
			}
			svg.WriteString(fmt.Sprintf(`<text x="%g" y="%g" class="%s" font-size="8" text-anchor="middle">%s</text>`,
				lp[0], lp[1], accent, esc(conn.Label)))
		}
	}

	// Components
	typeLabels := map[string]string{
		"frontend": "Frontend", "backend": "Backend", "database": "Database",
		"cloud": "Cloud", "security": "Security", "messagebus": "Message Bus", "external": "External",
	}
	for _, mc := range compMap {
		fill := "c-external"
		if f, ok := map[string]string{
			"frontend": "c-frontend", "backend": "c-backend", "database": "c-database",
			"cloud": "c-cloud", "security": "c-security", "messagebus": "c-messagebus", "external": "c-external",
		}[mc.Type]; ok {
			fill = f
		}

		labelY := mc.y + mc.height/2 + 4
		if mc.Sublabel != "" {
			labelY = mc.y + mc.height/2 - 2
		}

		svg.WriteString(fmt.Sprintf(`<rect x="%g" y="%g" width="%g" height="%g" rx="6" class="c-mask"/>`,
			mc.x, mc.y, mc.width, mc.height))
		svg.WriteString(fmt.Sprintf(`<rect x="%g" y="%g" width="%g" height="%g" rx="6" class="%s" stroke-width="1.5"/>`,
			mc.x, mc.y, mc.width, mc.height, fill))
		svg.WriteString(fmt.Sprintf(`<text x="%g" y="%g" class="t-primary" font-size="11" font-weight="600" text-anchor="middle">%s</text>`,
			mc.cx, labelY, esc(mc.Label)))

		if mc.Sublabel != "" {
			svg.WriteString(fmt.Sprintf(`<text x="%g" y="%g" class="t-muted" font-size="9" text-anchor="middle">%s</text>`,
				mc.cx, mc.y+mc.height/2+14, esc(mc.Sublabel)))
		}
		if mc.Tag != "" {
			svg.WriteString(fmt.Sprintf(`<text x="%g" y="%g" class="t-muted" font-size="7" text-anchor="middle">%s</text>`,
				mc.cx, mc.y+mc.height-8, esc(mc.Tag)))
		}
	}

	// Legend
	legendY := vbH - 16
	clear(used)
	legX := float64(margin)
	svg.WriteString(fmt.Sprintf(`<text x="%g" y="%g" class="t-primary" font-size="9" font-weight="600">Legend</text>`,
		legX, legendY-13))
	for _, mc := range compMap {
		if used[mc.Type] {
			continue
		}
		used[mc.Type] = true
		fill := "c-external"
		if f, ok := map[string]string{
			"frontend": "c-frontend", "backend": "c-backend", "database": "c-database",
			"cloud": "c-cloud", "security": "c-security", "messagebus": "c-messagebus", "external": "c-external",
		}[mc.Type]; ok {
			fill = f
		}
		label := typeLabels[mc.Type]
		if label == "" {
			label = mc.Type
		}
		svg.WriteString(fmt.Sprintf(`<rect x="%g" y="%g" width="14" height="9" rx="2" class="%s" stroke-width="1"/>`,
			legX, legendY-8, fill))
		svg.WriteString(fmt.Sprintf(`<text x="%g" y="%g" class="t-muted" font-size="8">%s</text>`,
			legX+20, legendY, esc(label)))
		legX += 30.0 + float64(len(label)*5) + 28.0
	}

	svg.WriteString(`</svg>`)
	return svg.String()
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// --- Workflow renderer (simplified) ---

func renderWorkflow(d *Diagram) string {
	var svg strings.Builder
	svg.WriteString(`<svg viewBox="0 0 800 600" role="img">`)
	svg.WriteString(renderDefs())

	// Compute viewBox from steps
	maxX, maxY := 800.0, 600.0
	for _, s := range asArray(d.Steps) {
		if s.Pos[0]+200 > maxX {
			maxX = s.Pos[0] + 200
		}
		if s.Pos[1]+80 > maxY {
			maxY = s.Pos[1] + 80
		}
	}
	svg.Reset()
	svg.WriteString(fmt.Sprintf(`<svg viewBox="0 0 %g %g" role="img">`, maxX+40, maxY+40))
	svg.WriteString(renderDefs())

	stepMap := make(map[string]Step)
	for _, s := range asArray(d.Steps) {
		stepMap[s.ID] = s
	}

	// Render transitions as arrows
	for _, t := range asArray(d.Transitions) {
		from, ok1 := stepMap[t.From]
		to, ok2 := stepMap[t.To]
		if !ok1 || !ok2 {
			continue
		}
		points := []Point{
			{from.Pos[0] + 80, from.Pos[1] + 30},
			{(from.Pos[0] + 80 + to.Pos[0] + 80) / 2, from.Pos[1] + 30},
			{(from.Pos[0] + 80 + to.Pos[0] + 80) / 2, to.Pos[1] + 30},
			{to.Pos[0], to.Pos[1] + 30},
		}
		svg.WriteString(fmt.Sprintf(`<path d="%s" class="a-default" stroke-width="1.5" marker-end="url(#arrowhead)"/>`,
			roundedPath(points, 8)))
		if t.Label != "" {
			lp := labelPoint(&Connection{LabelDx: 0, LabelDy: 0}, points)
			svg.WriteString(fmt.Sprintf(`<text x="%g" y="%g" class="t-muted" font-size="9" text-anchor="middle">%s</text>`,
				lp[0], lp[1], esc(t.Label)))
		}
	}

	// Render steps
	for _, s := range asArray(d.Steps) {
		cls := "c-backend"
		if s.Type == "start" || s.Type == "end" {
			cls = "c-frontend"
		} else if s.Type == "decision" {
			cls = "c-database"
		}
		rx := 6.0
		if s.Shape == "diamond" {
			rx = 0
		}
		svg.WriteString(fmt.Sprintf(`<rect x="%g" y="%g" width="160" height="60" rx="%g" class="%s" stroke-width="1.5"/>`,
			s.Pos[0], s.Pos[1], rx, cls))
		svg.WriteString(fmt.Sprintf(`<text x="%g" y="%g" class="t-primary" font-size="11" font-weight="600" text-anchor="middle">%s</text>`,
			s.Pos[0]+80, s.Pos[1]+34, esc(s.Label)))
	}

	svg.WriteString(`</svg>`)
	return svg.String()
}

// --- Sequence renderer (simplified) ---

func renderSequence(d *Diagram) string {
	var svg strings.Builder

	// Compute viewBox
	maxX := 600.0
	for _, p := range asArray(d.Participants) {
		if p.Pos[0]+100 > maxX {
			maxX = p.Pos[0] + 100
		}
	}

	partMap := make(map[string]SequenceParticipant)
	for _, p := range asArray(d.Participants) {
		partMap[p.ID] = p
	}

	y := 80.0
	stepY := 60.0

	svg.WriteString(fmt.Sprintf(`<svg viewBox="0 0 %g %g" role="img">`, maxX+100, float64(len(d.Messages)*60)+200))
	svg.WriteString(renderDefs())

	// Lifelines
	for _, p := range asArray(d.Participants) {
		svg.WriteString(fmt.Sprintf(`<line x1="%g" y1="60" x2="%g" y2="%g" class="c-grid" stroke-width="1" stroke-dasharray="4"/>`,
			p.Pos[0]+50, p.Pos[0]+50, float64(len(d.Messages)*60)+200))
		svg.WriteString(fmt.Sprintf(`<rect x="%g" y="20" width="100" height="40" rx="6" class="c-backend"/>`,
			p.Pos[0]))
		svg.WriteString(fmt.Sprintf(`<text x="%g" y="44" class="t-primary" font-size="10" font-weight="600" text-anchor="middle">%s</text>`,
			p.Pos[0]+50, esc(p.Label)))
	}

	// Messages
	for _, m := range asArray(d.Messages) {
		from, ok1 := partMap[m.From]
		to, ok2 := partMap[m.To]
		if !ok1 || !ok2 {
			continue
		}
		y += stepY
		x1, x2 := from.Pos[0]+50, to.Pos[0]+50
		cls := "a-default"
		if m.Type == "async" {
			cls = "a-dashed"
		} else if m.Type == "return" {
			cls = "a-emphasis"
		}
		svg.WriteString(fmt.Sprintf(`<line x1="%g" y1="%g" x2="%g" y2="%g" class="%s" stroke-width="1.5" marker-end="url(#arrowhead)"/>`,
			x1, y, x2, y, cls))
		midX := (x1 + x2) / 2
		svg.WriteString(fmt.Sprintf(`<text x="%g" y="%g" class="t-muted" font-size="9" text-anchor="middle">%s</text>`,
			midX, y-8, esc(m.Label)))
	}

	svg.WriteString(`</svg>`)
	return svg.String()
}

// --- Dataflow renderer (simplified) ---

func renderDataflow(d *Diagram) string {
	var svg strings.Builder
	maxX, maxY := 800.0, 600.0
	for _, n := range asArray(d.Nodes) {
		if n.Pos[0]+200 > maxX {
			maxX = n.Pos[0] + 200
		}
		if n.Pos[1]+80 > maxY {
			maxY = n.Pos[1] + 80
		}
	}

	svg.WriteString(fmt.Sprintf(`<svg viewBox="0 0 %g %g" role="img">`, maxX+40, maxY+40))
	svg.WriteString(renderDefs())

	nodeMap := make(map[string]FlowNode)
	for _, n := range asArray(d.Nodes) {
		nodeMap[n.ID] = n
	}

	// Edges
	for _, e := range asArray(d.Edges) {
		from, ok1 := nodeMap[e.From]
		to, ok2 := nodeMap[e.To]
		if !ok1 || !ok2 {
			continue
		}
		points := []Point{
			{from.Pos[0] + 80, from.Pos[1] + 30},
			{(from.Pos[0] + 80 + to.Pos[0] + 80) / 2, (from.Pos[1] + 30 + to.Pos[1] + 30) / 2},
			{to.Pos[0], to.Pos[1] + 30},
		}
		svg.WriteString(fmt.Sprintf(`<path d="%s" class="a-default" stroke-width="1.5" marker-end="url(#arrowhead)"/>`,
			roundedPath(points, 8)))
		if e.Label != "" {
			lp := labelPoint(&Connection{LabelDx: 0, LabelDy: 0}, points)
			svg.WriteString(fmt.Sprintf(`<text x="%g" y="%g" class="t-muted" font-size="9" text-anchor="middle">%s</text>`,
				lp[0], lp[1], esc(e.Label)))
		}
	}

	// Nodes
	for _, n := range asArray(d.Nodes) {
		cls := "c-backend"
		if n.Type == "source" {
			cls = "c-frontend"
		} else if n.Type == "store" {
			cls = "c-database"
		} else if n.Type == "sink" {
			cls = "c-external"
		}
		svg.WriteString(fmt.Sprintf(`<rect x="%g" y="%g" width="160" height="60" rx="6" class="%s" stroke-width="1.5"/>`,
			n.Pos[0], n.Pos[1], cls))
		svg.WriteString(fmt.Sprintf(`<text x="%g" y="%g" class="t-primary" font-size="11" font-weight="600" text-anchor="middle">%s</text>`,
			n.Pos[0]+80, n.Pos[1]+34, esc(n.Label)))
	}

	svg.WriteString(`</svg>`)
	return svg.String()
}

// --- Lifecycle renderer (simplified) ---

func renderLifecycle(d *Diagram) string {
	// Lifecycle is essentially a workflow with different styling
	return renderWorkflow(d)
}

// --- Shared SVG defs ---

func renderDefs() string {
	return `<defs>
<marker id="arrowhead" markerWidth="10" markerHeight="7" refX="9" refY="3.5" orient="auto"><polygon points="0 0, 10 3.5, 0 7" class="m-default"/></marker>
<marker id="arrowhead-emphasis" markerWidth="10" markerHeight="7" refX="9" refY="3.5" orient="auto"><polygon points="0 0, 10 3.5, 0 7" class="m-emphasis"/></marker>
<marker id="arrowhead-security" markerWidth="10" markerHeight="7" refX="9" refY="3.5" orient="auto"><polygon points="0 0, 10 3.5, 0 7" class="m-security"/></marker>
<marker id="arrowhead-dashed" markerWidth="10" markerHeight="7" refX="9" refY="3.5" orient="auto"><polygon points="0 0, 10 3.5, 0 7" class="m-dashed"/></marker>
<pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse"><path d="M 40 0 L 0 0 0 40" class="c-grid" stroke-width="0.5"/></pattern>
</defs>`
}

func renderCards(cards []Card) string {
	if len(cards) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<div class="cards">`)
	for _, card := range cards {
		b.WriteString(`<div class="card"><div class="card-header"><div class="card-dot `)
		b.WriteString(esc(card.Dot))
		b.WriteString(`"></div><h3>`)
		b.WriteString(esc(card.Title))
		b.WriteString(`</h3></div><ul>`)
		for _, item := range card.Items {
			b.WriteString(`<li>&bull; `)
			b.WriteString(esc(item))
			b.WriteString(`</li>`)
		}
		b.WriteString(`</ul></div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

// --- HTML template ---

func applyTemplate(meta Meta, svg, cards, diagType string) string {
	title := esc(meta.Title)
	subtitle := esc(meta.Subtitle)
	footer := fmt.Sprintf("%s diagram &bull; Built with Talon <span class=\"no-print\">&bull; Press <kbd>T</kbd> for theme and <kbd>E</kbd> for export</span>", strings.Title(string(diagType)))

	return `<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
<meta name="generator" content="talon 1.0"/>
<title>` + title + ` Diagram</title>
<script>
(function(){try{var t=null;try{var p=new URLSearchParams(window.location.search).get('theme');if(p==='light'||p==='dark')t=p}catch(_){}if(!t){try{t=localStorage.getItem('archify-theme')}catch(_){}}if(t!=='light'&&t!=='dark'){t=window.matchMedia('(prefers-color-scheme:light)').matches?'light':'dark'}document.documentElement.setAttribute('data-theme',t)}catch(_){}})()
</script>
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet" media="print" onload="this.media='all'">
<noscript><link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet"></noscript>
<style>
:root,[data-theme="dark"]{--bg:#020617;--grid:#1e293b;--text:#fff;--text-muted:#94a3b8;--text-dim:#475569;--text-faint:#7d8da1;--panel:rgba(15,23,42,.5);--panel-border:#1e293b;--lane-fill:rgba(15,23,42,.22);--lane-stroke:#334155;--arrow:#64748b;--arrow-emphasis:#34d399;--mask:#0f172a;--frontend-fill:rgba(8,51,68,.4);--frontend-stroke:#22d3ee;--backend-fill:rgba(6,78,59,.4);--backend-stroke:#34d399;--database-fill:rgba(76,29,149,.4);--database-stroke:#a78bfa;--cloud-fill:rgba(120,53,15,.3);--cloud-stroke:#fbbf24;--security-fill:rgba(136,19,55,.4);--security-stroke:#fb7185;--messagebus-fill:rgba(251,146,60,.3);--messagebus-stroke:#fb923c;--external-fill:rgba(30,41,59,.5);--external-stroke:#94a3b8;--toolbar-bg:rgba(15,23,42,.8);--toolbar-border:#334155;--toolbar-text:#e2e8f0;--toolbar-hover:rgba(15,23,42,.95);--toolbar-menu-bg:#0f172a}
[data-theme="light"]{--bg:#f8fafc;--grid:#e2e8f0;--text:#0f172a;--text-muted:#64748b;--text-dim:#94a3b8;--text-faint:#64748b;--panel:#fff;--panel-border:#e2e8f0;--lane-fill:rgba(248,250,252,.65);--lane-stroke:#cbd5e1;--arrow:#94a3b8;--arrow-emphasis:#059669;--mask:#fff;--frontend-fill:rgba(34,211,238,.15);--frontend-stroke:#0891b2;--backend-fill:rgba(52,211,153,.18);--backend-stroke:#059669;--database-fill:rgba(167,139,250,.2);--database-stroke:#7c3aed;--cloud-fill:rgba(251,191,36,.18);--cloud-stroke:#d97706;--security-fill:rgba(251,113,133,.15);--security-stroke:#e11d48;--messagebus-fill:rgba(251,146,60,.15);--messagebus-stroke:#ea580c;--external-fill:rgba(148,163,184,.18);--external-stroke:#64748b;--toolbar-bg:rgba(255,255,255,.92);--toolbar-border:#cbd5e1;--toolbar-text:#334155;--toolbar-hover:#fff;--toolbar-menu-bg:#fff}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'JetBrains Mono',ui-monospace,SFMono-Regular,Menlo,Consolas,'DejaVu Sans Mono','Liberation Mono','Noto Sans Mono CJK SC','PingFang SC','Hiragino Sans GB','Microsoft YaHei',monospace;background:var(--bg);min-height:100vh;padding:2rem;color:var(--text);transition:background .2s ease,color .2s ease}
.container{max-width:1200px;margin:0 auto}
.header{margin-bottom:2rem}
.header-row{display:flex;align-items:center;gap:1rem;margin-bottom:.5rem}
.pulse-dot{width:12px;height:12px;background:var(--frontend-stroke);border-radius:50%;animation:pulse 2s infinite}
@keyframes pulse{0%,100%{opacity:1}50%{opacity:.5}}
h1{font-size:1.5rem;font-weight:700;letter-spacing:-.025em}
.subtitle{color:var(--text-muted);font-size:.875rem;margin-left:1.75rem}
.diagram-container{background:var(--panel);border-radius:1rem;border:1px solid var(--panel-border);padding:1.5rem;overflow-x:auto}
svg{width:100%;min-width:min(900px,100%);display:block}
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:1rem;margin-top:2rem}
.card{background:var(--panel);border-radius:.75rem;border:1px solid var(--panel-border);padding:1.25rem}
.card-header{display:flex;align-items:center;gap:.5rem;margin-bottom:.75rem}
.card-dot{width:8px;height:8px;border-radius:50%}
.card-dot.cyan{background:var(--frontend-stroke)}.card-dot.emerald{background:var(--backend-stroke)}.card-dot.violet{background:var(--database-stroke)}.card-dot.amber{background:var(--cloud-stroke)}.card-dot.rose{background:var(--security-stroke)}.card-dot.orange{background:var(--messagebus-stroke)}.card-dot.slate{background:var(--external-stroke)}
.card h3{font-size:.875rem;font-weight:600;color:var(--text)}
.card ul{list-style:none;color:var(--text-muted);font-size:.75rem}
.card li{margin-bottom:.375rem}
.footer{text-align:center;margin-top:1.5rem;color:var(--text-faint);font-size:.75rem}
@page{size:landscape;margin:1.5cm}
@media print{:root,[data-theme="dark"],[data-theme="light"]{--bg:#fff;--grid:transparent;--panel:#fff;--panel-border:#e2e8f0;--text:#0f172a;--text-muted:#475569;--text-dim:#94a3b8;--text-faint:#64748b;--mask:#fff;--lane-fill:rgba(248,250,252,.65);--lane-stroke:#cbd5e1;--arrow:#94a3b8;--arrow-emphasis:#059669;--frontend-fill:rgba(34,211,238,.15);--frontend-stroke:#0891b2;--backend-fill:rgba(52,211,153,.18);--backend-stroke:#059669;--database-fill:rgba(167,139,250,.2);--database-stroke:#7c3aed;--cloud-fill:rgba(251,191,36,.18);--cloud-stroke:#d97706;--security-fill:rgba(251,113,133,.15);--security-stroke:#e11d48;--messagebus-fill:rgba(251,146,60,.15);--messagebus-stroke:#ea580c;--external-fill:rgba(148,163,184,.18);--external-stroke:#64748b}body{background:#fff!important;padding:0}.container{max-width:none}.toolbar,.archify-toast,.no-print{display:none!important}.diagram-container,.card{box-shadow:none;border-color:#e2e8f0}.cards{grid-template-columns:1fr 1fr}.card{break-inside:avoid;page-break-inside:avoid}.diagram-container{break-inside:avoid;page-break-inside:avoid}h1,.subtitle{color:#0f172a}}
.toolbar{position:fixed;top:1rem;right:1rem;display:flex;gap:.5rem;z-index:100}
.toolbar button{background:var(--toolbar-bg);color:var(--toolbar-text);border:1px solid var(--toolbar-border);padding:.5rem .875rem;border-radius:.5rem;font-family:inherit;font-size:.75rem;font-weight:500;cursor:pointer;backdrop-filter:blur(8px);transition:background .15s,border-color .15s;display:inline-flex;align-items:center;gap:.375rem}
.toolbar button:hover{background:var(--toolbar-hover);border-color:var(--arrow)}
.toolbar button:focus-visible{outline:2px solid var(--frontend-stroke);outline-offset:2px}
.toolbar .export-wrap{position:relative}
.toolbar .export-menu{position:absolute;top:calc(100% + .375rem);right:0;background:var(--toolbar-menu-bg);border:1px solid var(--toolbar-border);border-radius:.5rem;padding:.375rem;min-width:160px;display:none;flex-direction:column;box-shadow:0 4px 20px rgba(0,0,0,.25)}
.toolbar .export-menu.open{display:flex}
.toolbar .export-menu button{background:transparent;border:0;justify-content:space-between;width:100%;text-align:left;padding:.5rem .75rem}
.toolbar .export-menu button:hover{background:var(--toolbar-hover)}
.toolbar .export-menu .hint{color:var(--text-faint);font-size:.625rem}
.toolbar .export-menu hr{border:0;border-top:1px solid var(--toolbar-border);margin:.25rem .375rem;opacity:.5}
.archify-toast{position:fixed;top:1rem;left:50%;transform:translateX(-50%) translateY(-8px);background:var(--toolbar-menu-bg);color:var(--text);border:1px solid var(--toolbar-border);padding:.5rem 1rem;border-radius:.5rem;font-family:inherit;font-size:.75rem;opacity:0;transition:opacity .2s,transform .2s;z-index:200;pointer-events:none;box-shadow:0 4px 20px rgba(0,0,0,.25)}
.archify-toast.show{opacity:1;transform:translateX(-50%) translateY(0)}
/* SVG classes */
.c-grid{stroke:var(--grid);fill:none}
.c-mask{fill:var(--mask);stroke:none}
.c-frontend{fill:var(--frontend-fill);stroke:var(--frontend-stroke)}
.c-backend{fill:var(--backend-fill);stroke:var(--backend-stroke)}
.c-database{fill:var(--database-fill);stroke:var(--database-stroke)}
.c-cloud{fill:var(--cloud-fill);stroke:var(--cloud-stroke)}
.c-security{fill:var(--security-fill);stroke:var(--security-stroke)}
.c-messagebus{fill:var(--messagebus-fill);stroke:var(--messagebus-stroke)}
.c-external{fill:var(--external-fill);stroke:var(--external-stroke)}
.t-primary{fill:var(--text)}
.t-muted{fill:var(--text-muted)}
.t-dim{fill:var(--text-dim)}
.t-frontend{fill:var(--frontend-stroke)}
.t-backend{fill:var(--backend-stroke)}
.t-database{fill:var(--database-stroke)}
.t-cloud{fill:var(--cloud-stroke)}
.t-security{fill:var(--security-stroke)}
.t-messagebus{fill:var(--messagebus-stroke)}
.t-external{fill:var(--external-stroke)}
.a-default{stroke:var(--arrow);fill:none}
.a-emphasis{stroke:var(--arrow-emphasis);fill:none}
.a-security{stroke:var(--security-stroke);fill:none;stroke-dasharray:5,5}
.a-dashed{stroke:var(--database-stroke);fill:none;stroke-dasharray:4,4}
.m-default{fill:var(--arrow)}
.m-emphasis{fill:var(--arrow-emphasis)}
.m-security{fill:var(--security-stroke)}
.m-dashed{fill:var(--database-stroke)}
.c-security-group{fill:transparent;stroke:var(--security-stroke);stroke-dasharray:4,4}
.c-region{fill:rgba(251,191,36,.05);stroke:var(--cloud-stroke);stroke-dasharray:8,4}
@media(max-width:720px){body{padding:1rem}.toolbar{position:static;justify-content:flex-end;margin-bottom:1rem}.header{margin-bottom:1.25rem}.header-row{gap:.75rem;align-items:flex-start}h1{font-size:1.25rem;line-height:1.25}.subtitle{margin-left:0;font-size:.8125rem;line-height:1.55}.diagram-container{padding:.75rem;border-radius:.75rem}.cards{grid-template-columns:1fr;margin-top:1rem}}
</style>
</head>
<body>
<div class="toolbar" role="toolbar" aria-label="Diagram actions">
<button id="btn-theme" type="button" title="Toggle theme (T)" aria-label="Toggle color theme" aria-pressed="false"><span id="theme-icon" aria-hidden="true">&#9790;</span><span id="theme-label">Dark</span></button>
<div class="export-wrap"><button id="btn-export" type="button" title="Export diagram (E)" aria-label="Export diagram" aria-haspopup="menu" aria-expanded="false" aria-controls="export-menu">Export <span aria-hidden="true">&#9662;</span></button>
<div class="export-menu" id="export-menu" role="menu" aria-label="Export">
<button data-action="copy" type="button" role="menuitem" tabindex="-1">Copy to clipboard <span class="hint">PNG</span></button>
<hr role="separator">
<button data-format="png" type="button" role="menuitem" tabindex="-1">Download PNG</button>
<button data-format="jpeg" type="button" role="menuitem" tabindex="-1">Download JPEG</button>
<button data-format="webp" type="button" role="menuitem" tabindex="-1">Download WebP</button>
<hr role="separator">
<button data-format="svg" type="button" role="menuitem" tabindex="-1">Download SVG <span class="hint">vector</span></button>
</div></div></div>
<div class="container">
<div class="header"><div class="header-row"><div class="pulse-dot"></div><h1>` + title + `</h1></div><p class="subtitle">` + subtitle + `</p></div>
<div class="diagram-container">
` + svg + `
</div>
` + cards + `
<p class="footer">` + footer + `</p>
</div>
<div id="archify-toast" class="archify-toast" role="status" aria-live="polite"></div>
<script>
(function(){var e=document.getElementById('btn-theme'),m=document.getElementById('btn-export'),c=document.getElementById('export-menu'),t=document.getElementById('archify-toast'),r=function(u){t.textContent=u;t.classList.add('show');setTimeout(function(){t.classList.remove('show')},2000)};function l(){var n=document.documentElement.getAttribute('data-theme');n=n==='light'?'dark':'light';document.documentElement.setAttribute('data-theme',n);try{localStorage.setItem('archify-theme',n)}catch(_){}e.querySelector('#theme-label').textContent=n==='dark'?'Dark':'Light';e.setAttribute('aria-pressed',n==='light'?'true':'false')}function v(){c.classList.toggle('open');m.setAttribute('aria-expanded',c.classList.contains('open')?'true':'false')}function x(f){var s=document.querySelector('svg');if(!s)return;if(f==='svg'){var a=document.createElement('a');a.href='data:image/svg+xml,'+encodeURIComponent(new XMLSerializer().serializeToString(s));a.download='diagram.svg';a.click();r('SVG downloaded')}else{var b=new XMLSerializer().serializeToString(s),d=new Blob([b],{type:'image/svg+xml;charset=utf-8'}),u=URL.createObjectURL(d),i=new Image;i.onload=function(){var g=document.createElement('canvas');g.width=i.naturalWidth*2;g.height=i.naturalHeight*2;var k=g.getContext('2d');k.scale(2,2);k.drawImage(i,0,0);g.toBlob(function(blob){if(f==='copy'){navigator.clipboard.write([new ClipboardItem({'image/png':blob})]).then(function(){r('Copied to clipboard')}).catch(function(){r('Copy failed')})}else{var a=document.createElement('a');a.href=URL.createObjectURL(blob);a.download='diagram.'+f;a.click();r(f.toUpperCase()+' downloaded')}},'image/'+f,1)};i.src=u}}e.addEventListener('click',l);m.addEventListener('click',v);document.addEventListener('keydown',function(n){if(n.key==='t'||n.key==='T'){l();n.preventDefault()}if(n.key==='e'||n.key==='E'){v();n.preventDefault()}if(n.key==='Escape')c.classList.remove('open')});c.querySelectorAll('button').forEach(function(b){b.addEventListener('click',function(){var f=b.getAttribute('data-format')||b.getAttribute('data-action');if(f==='copy')x('copy');else if(f)x(f);c.classList.remove('open');m.setAttribute('aria-expanded','false')})});document.addEventListener('click',function(n){if(!n.target.closest('.export-wrap')){c.classList.remove('open');m.setAttribute('aria-expanded','false')}})
})();
</script>
</body>
</html>`
}
