---
name: talon-viz
description: Create professional architecture, workflow, sequence, data-flow, and lifecycle/state diagrams as standalone HTML files with SVG graphics, a built-in dark/light theme toggle, and one-click export to PNG / JPEG / WebP / SVG. Accepts JSON intermediate representation and renders it through Go-based renderers. Use when the user asks for system architecture diagrams, infrastructure diagrams, cloud architecture visualizations, security diagrams, network topology, technical workflows, approval flows, runbooks, CI/CD flows, process diagrams, API call sequences, request lifecycles, data pipelines, ETL/ELT maps, PII boundaries, data lineage, state machines, lifecycle diagrams, status transitions.
license: MIT
metadata:
  version: "1.0"
---

# Talon Viz Skill

Generate professional technical diagrams as standalone HTML files — zero dependencies, works in any browser. Every diagram ships with inline SVG, a dark/light theme toggle (persists in `localStorage`, respects `prefers-color-scheme`), and a built-in export menu (copy PNG to clipboard, download PNG/JPEG/WebP at 2× resolution, download dual-theme SVG).

## How It Works

The `deps/viz` Go package produces diagrams from a JSON intermediate representation (IR). You construct a `viz.Diagram` struct (or write the JSON by hand), call `viz.Generate()`, and get a self-contained HTML file.

```
JSON IR ──→ viz.Generate() ──→ standalone HTML
```

No build step, no npm install, no runtime dependencies. The generated HTML works offline in any browser.

## Diagram Types

| Type | Use for |
|------|---------|
| `architecture` | System components, services, databases, caches, security boundaries, infrastructure |
| `workflow` | Technical flows, approval gates, CI/CD, runbooks, incident response |
| `sequence` | API call chains, request lifecycles, cache fallback, async traces |
| `dataflow` | Data pipelines, ETL/ELT, PII boundaries, lineage, consumers |
| `lifecycle` | State machines, status transitions, wait states, retries, terminal outcomes |

The JSON schema for each type lives in `schemas/<type>.schema.json`. Worked examples are in `examples/*.json` with rendered HTML outputs in `examples/results/`.

## Creating a Diagram

### Architecture (most common)

Construct a `viz.Diagram` with `TypeArchitecture`, add `Components`, `Boundaries`, and `Connections`:

```go
d := &viz.Diagram{
    SchemaVersion: 1,
    DiagramType:   viz.TypeArchitecture,
    Meta: viz.Meta{
        Title:    "My App",
        Subtitle: "React + Express + PostgreSQL",
    },
    Components: []viz.Component{
        {ID: "frontend", Type: "frontend", Label: "React SPA", Pos: [2]float64{40, 40}},
        {ID: "api",      Type: "backend",  Label: "Express API", Pos: [2]float64{40, 200}},
        {ID: "db",       Type: "database", Label: "PostgreSQL",  Pos: [2]float64{40, 400], Size: &[2]float64{160, 70}},
    },
    Connections: []viz.Connection{
        {From: "frontend", To: "api", Label: "HTTPS", Variant: "emphasis"},
        {From: "api",      To: "db",  Label: "SQL"},
    },
}

html, err := viz.Generate(d)
os.WriteFile("diagram.html", html, 0644)
```

Components use **free coordinates** — you choose `pos: [x, y]` as top-left and optional `size: [w, h]` (defaults to 120×60). The viewBox auto-fits to your components plus a legend row.

#### Component Types (color mapping)

| Type | Color | Stroke |
|------|-------|--------|
| `frontend` | Cyan `#22d3ee` | Client apps, UI |
| `backend` | Emerald `#34d399` | APIs, services, workers |
| `database` | Violet `#a78bfa` | Databases, caches, stores |
| `cloud` | Amber `#fbbf24` | Managed services, CDN, LB |
| `security` | Rose `#fb7185` | Auth, JWT, security groups |
| `messagebus` | Orange `#fb923c` | Kafka, SQS, event buses |
| `external` | Slate `#94a3b8` | Users, 3rd parties, generic |

#### Boundaries

Group components inside a `region` (dashed amber) or `security-group` (dashed rose). The renderer computes the bounding box automatically:

```go
Boundaries: []viz.Boundary{
    {Kind: "region",         Label: "AWS", Wraps: []string{"api", "db"}},
    {Kind: "security-group", Label: "sg-api", Wraps: []string{"api"}},
}
```

#### Connections

Connections route between components with automatic orthogonal routing. Control with `FromSide`/`ToSide`, `Variant`, and label offsets:

```go
Connections: []viz.Connection{
    {From: "users", To: "cdn", Label: "HTTPS", Variant: "emphasis"},
    {From: "cdn",   To: "api", Label: "proxy"},
    {From: "api",   To: "cache", Label: "cache-aside", Variant: "dashed", LabelDy: -20},
}
```

| Variant | Style | Use |
|---------|-------|-----|
| `default` | Solid gray | Standard calls |
| `emphasis` | Solid green | Primary/hot path |
| `security` | Dashed rose | Auth, PII, policy |
| `dashed` | Dashed violet | Async, batch, cache |

Labels auto-center on the connection midpoint. Adjust with `LabelDx`/`LabelDy` (pixel offsets) or `LabelSegment` to pick a different polyline segment.

### Other Diagram Types

The same JSON IR structure applies. Set the appropriate `DiagramType`:

- `TypeWorkflow` — uses `Steps` + `Transitions` with lane-based layout
- `TypeSequence` — uses `Participants` + `Messages` with lifelines
- `TypeDataflow` — uses `Nodes` + `Edges` with stage-based layout
- `TypeLifecycle` — uses `Steps` + `Transitions` (workflow-derived)

See `schemas/*.json` for the exact field shapes and `examples/*.json` for worked data.

## Summary Cards

Add context below the diagram with `Cards`. Each card has a colored dot, title, and bullet items:

```go
Cards: []viz.Card{
    {Dot: "violet", Title: "Routing", Items: []string{
        "Static assets from CDN",
        "API proxied to Express",
    }},
    {Dot: "rose", Title: "Security", Items: []string{
        "JWT on every protected route",
        "Token blacklist in Redis",
    }},
}
```

Dot colors: `cyan`, `emerald`, `violet`, `amber`, `rose`, `orange`, `slate` — they map to the component type strokes.

## Layout Tips

- **Avoid connection crossings** — place components so arrows flow without intersecting. A 3-row × 2-column grid (frontend top-left, data stores bottom-right) is a reliable default.
- **Push labels apart** — use `LabelDx`/`LabelDy` to nudge overlapping labels. A `LabelDy: -12` on a horizontal connection lifts its label above the arrow.
- **Boundary headroom** — boundaries auto-pad with 30px on top/left/right and 50px on the bottom. Wrapping components that span the full diagram width is fine.
- **Legend overflow** — the viewBox auto-widens to fit the legend row, so all component type labels are visible.
- **Component overlap** — the validator checks for components less than 8px apart and connection labels that overlap components.
- **Multi-line components** — add `Sublabel` (rendered below the label, smaller) and `Tag` (rendered at the bottom, muted) for extra context.

## Reference Files

| Path | Purpose |
|------|---------|
| `schemas/*.json` | JSON Schema for each diagram type |
| `examples/*.json` | Worked input files for each type |
| `examples/results/*.html` | Rendered output for each example |
| `renderers/*/*.mjs` | JavaScript reference renderers (archify origin) |
| `assets/template.html` | Standalone HTML template (archify origin) |
| `viz.go` | Go implementation — all renderers + template |
