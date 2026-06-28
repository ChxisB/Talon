# Workflow Renderer (Reference — JS)

Render `diagram_type: "workflow"` JSON files into the standalone HTML template.
This is a **JavaScript reference implementation** alongside the primary Go
renderer in `deps/viz/viz.go`. Both consume the same JSON schema and produce
identical output.

```bash
node renderers/workflow/render-workflow.mjs input.workflow.json output.html
```

The JS renderer validates input against `schemas/workflow.schema.json` via ajv.
Install deps with `npm install` from the viz folder root. Without it, the
renderer prints a warning and skips schema validation; its own layout checks
still run.

If `output.html` is omitted, the renderer uses `meta.output` from the JSON file
or falls back to `workflow.html`.

> **Note**: For new diagrams, prefer the Go renderer (`viz.Generate()` in Go
> code). The JS renderers exist for reference and schema validation.

## Input

Workflow JSON files must set:

```json
{
  "schema_version": 1,
  "diagram_type": "workflow",
  "meta": {
    "title": "Agent Tool Call Workflow",
    "subtitle": "Renderer-driven workflow prototype"
  },
  "lanes": [],
  "nodes": [],
  "edges": [],
  "cards": []
}
```

Omit `meta.viewBox` for the common case: the width is fixed at 720 and the
height is derived from the lane count, so lanes and legend always fit. A
complete worked example lives at `examples/agent-tool-call.workflow.json`.

The schema lives at `schemas/workflow.schema.json`.

## Layout budget

| Constant | Value |
|----------|-------|
| viewBox | default `[720, auto]` — auto height = 52 + lanes×104 + (lanes−1)×20 + 124 |
| Lane frame | x 40, width 640, height 104, gap 20; first lane top at y 52 |
| Lane title strip | top 30px of each lane; node boxes must stay below it |
| Column centers (`col` 0–5) | x = 88, 220, 300, 430, 500, 625 |
| Default node | 92×52 (height 68 when `tag` is set) |
| Node spacing | ≥8px between nodes in the same lane |
| Edge length | straight segments must span ≥28px |
| Legend row | y = lane bottom + 44; viewBox height must be ≥ legend y + 18 |

Column-center gaps are 132 / 80 / 130 / 70 / 125 px: columns 1↔2 (80px) and
3↔4 (70px) cannot both hold default-width 92px nodes in the same lane — skip a
column or reduce `width`.

## Design Rules

- Use lanes for ownership or runtime boundaries.
- Place nodes with lane IDs and column indexes, not raw SVG coordinates.
- Leave short adjacent links unlabeled; the arrow is enough.
- Use labels for cross-lane decisions, approvals, async traces, and return paths.
- Prefer route presets — `drop` (bend between lanes; `bias` 0–1 picks where),
  `outside-right`, `return-left`, `bottom-channel`, and `up-channel` — before
  using raw `via` points. `straight` and the default `auto` cover the rest.
- Keep workflow examples compact enough to render well in narrow chat/browser
  previews.

Schema violations exit non-zero with path-prefixed messages annotated with the
element's id or label. The renderer additionally fails when it can detect
layout problems, including node overlap, nodes outside their lanes, unknown
edge targets, labels colliding with nodes or other labels, labels wider than
their node, legends outside the viewBox, or straight arrows that are too short
to read cleanly. Text width is estimated CJK-aware: fullwidth glyphs count as
two units.
