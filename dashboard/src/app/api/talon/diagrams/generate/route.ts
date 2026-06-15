import { NextRequest, NextResponse } from "next/server";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const VALID_TYPES = ["architecture", "workflow", "sequence", "dataflow", "lifecycle"] as const;
type DiagramType = (typeof VALID_TYPES)[number];

/**
 * POST /api/talon/diagrams/generate
 *
 * Body: { type: "architecture"|"workflow"|"sequence"|"dataflow"|"lifecycle", json: string }
 * Returns: { html: string }
 *
 * Generates a self-contained HTML diagram from JSON input.
 */
export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const { type, json } = body as { type: string; json: string };

    if (!type || !json) {
      return NextResponse.json(
        { error: "Missing required fields: type, json" },
        { status: 400 }
      );
    }

    if (!VALID_TYPES.includes(type as DiagramType)) {
      return NextResponse.json(
        { error: `Invalid type: ${type}. Valid: ${VALID_TYPES.join(", ")}` },
        { status: 400 }
      );
    }

    const html = generateDiagram(type as DiagramType, json);

    return NextResponse.json({ html });
  } catch (err: any) {
    return NextResponse.json(
      { error: `Diagram generation failed: ${err.message || String(err)}` },
      { status: 500 }
    );
  }
}

function generateDiagram(type: DiagramType, jsonStr: string): string {
  let data: any;
  try {
    data = JSON.parse(jsonStr);
  } catch {
    return errorHtml("Invalid JSON input");
  }

  switch (type) {
    case "architecture": return generateArchitecture(data);
    case "workflow":     return generateWorkflow(data);
    case "sequence":     return generateSequence(data);
    case "dataflow":     return generateDataflow(data);
    case "lifecycle":    return generateLifecycle(data);
    default: return errorHtml("Unknown diagram type");
  }
}

function errorHtml(msg: string): string {
  return `<!DOCTYPE html><html><body style="font-family:sans-serif;padding:40px;color:#e74c3c"><h2>Error</h2><p>${msg}</p></body></html>`;
}

function baseHtml(title: string, body: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>${title}</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; padding: 32px; background: #f8f9fa; color: #1a1a2e; }
  h1 { font-size: 24px; margin-bottom: 24px; color: #1a1a2e; }
  .container { max-width: 1200px; margin: 0 auto; }
  svg { background: white; border-radius: 12px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
  .legend { margin-top: 16px; display: flex; gap: 16px; flex-wrap: wrap; }
  .legend-item { display: flex; align-items: center; gap: 6px; font-size: 13px; color: #666; }
  .legend-item swatch { width: 12px; height: 12px; border-radius: 3px; }
</style></head>
<body><div class="container">${body}</div></body></html>`;
}

function generateArchitecture(data: any): string {
  const components = data.components ?? [];
  const connections = data.connections ?? [];
  const boundaries = data.boundaries ?? [];
  const width = 900;
  const height = 600;
  const boxW = 160;
  const boxH = 60;

  // Position components in a grid if no positions specified
  components.forEach((c: any, i: number) => {
    if (!c.pos) {
      const col = i % 4;
      const row = Math.floor(i / 4);
      c.pos = { x: 60 + col * 200, y: 80 + row * 120 };
    }
  });

  let svg = `<svg width="${width}" height="${height}" xmlns="http://www.w3.org/2000/svg">`;
  svg += `<defs>
    <marker id="arrow" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L10,5 L0,10" fill="#94a3b8"/></marker>
  </defs>`;

  // Background
  svg += `<rect width="${width}" height="${height}" fill="white" rx="8"/>`;

  // Boundaries
  boundaries.forEach((b: any) => {
    svg += `<rect x="${b.x}" y="${b.y}" width="${b.w}" height="${b.h}" fill="#f1f5f9" stroke="#cbd5e1" stroke-width="1.5" stroke-dasharray="6,3" rx="8"/>`;
    if (b.label) svg += `<text x="${b.x + 12}" y="${b.y + 24}" font-size="13" fill="#64748b" font-weight="600">${esc(b.label)}</text>`;
  });

  // Connections
  connections.forEach((conn: any) => {
    const from = components.find((c: any) => c.id === conn.from);
    const to = components.find((c: any) => c.id === conn.to);
    if (from && to) {
      const x1 = from.pos.x + boxW / 2;
      const y1 = from.pos.y + boxH;
      const x2 = to.pos.x + boxW / 2;
      const y2 = to.pos.y;
      svg += `<line x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}" stroke="#94a3b8" stroke-width="1.5" marker-end="url(#arrow)"/>`;
      if (conn.label) {
        const mx = (x1 + x2) / 2;
        const my = (y1 + y2) / 2;
        svg += `<rect x="${mx - 40}" y="${my - 10}" width="80" height="20" fill="white" rx="4"/>`;
        svg += `<text x="${mx}" y="${my + 4}" font-size="11" fill="#64748b" text-anchor="middle">${esc(conn.label)}</text>`;
      }
    }
  });

  // Components
  components.forEach((c: any) => {
    const color = typeColor(c.type || "service");
    svg += `<rect x="${c.pos.x}" y="${c.pos.y}" width="${boxW}" height="${boxH}" fill="${color.bg}" stroke="${color.border}" stroke-width="1.5" rx="8"/>`;
    svg += `<text x="${c.pos.x + boxW / 2}" y="${c.pos.y + 26}" font-size="13" fill="${color.text}" font-weight="600" text-anchor="middle">${esc(c.label || c.id)}</text>`;
    if (c.type) svg += `<text x="${c.pos.x + boxW / 2}" y="${c.pos.y + boxH - 12}" font-size="10" fill="${color.text}" opacity="0.6" text-anchor="middle">${esc(c.type)}</text>`;
  });

  svg += "</svg>";

  let legendHtml = "";
  const typeSet = new Set<string>();
  components.forEach((c: any) => typeSet.add(c.type || "service"));
  const usedTypes = [...typeSet];
  if (usedTypes.length > 1) {
    legendHtml = `<div class="legend">${usedTypes.map((t) => {
      const color = typeColor(t);
      return `<div class="legend-item"><swatch style="background:${color.bg};border:1px solid ${color.border}"></swatch>${t}</div>`;
    }).join("")}</div>`;
  }

  return baseHtml("Architecture Diagram",
    `<h1>Architecture Diagram</h1>${svg}${legendHtml}`
  );
}

function generateWorkflow(data: any): string {
  const steps = data.steps ?? [];
  const width = 800;
  const height = Math.max(200, steps.length * 100 + 80);
  const stepH = 50;

  let svg = `<svg width="${width}" height="${height}" xmlns="http://www.w3.org/2000/svg">`;
  svg += `<rect width="${width}" height="${height}" fill="white" rx="8"/>`;
  svg += `<defs>
    <marker id="arrow-down" viewBox="0 0 10 10" refX="5" refY="10" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L5,10 L10,0" fill="#94a3b8"/></marker>
  </defs>`;

  steps.forEach((step: any, i: number) => {
    const y = 60 + i * 80;
    const cx = width / 2;
    const color = statusColor(step.status || "pending");

    svg += `<rect x="${cx - 120}" y="${y}" width="240" height="${stepH}" fill="${color.bg}" stroke="${color.border}" stroke-width="1.5" rx="8"/>`;
    svg += `<text x="${cx}" y="${y + stepH / 2 + 4}" font-size="13" fill="${color.text}" font-weight="600" text-anchor="middle">${esc(step.name || `Step ${i + 1}`)}</text>`;

    if (step.description) {
      svg += `<text x="${cx}" y="${y + stepH + 16}" font-size="11" fill="#94a3b8" text-anchor="middle">${esc(step.description)}</text>`;
    }

    // Arrow between steps
    if (i < steps.length - 1) {
      svg += `<line x1="${cx}" y1="${y + stepH}" x2="${cx}" y2="${y + 80}" stroke="#cbd5e1" stroke-width="1.5" marker-end="url(#arrow-down)"/>`;
    }
  });

  svg += "</svg>";

  return baseHtml("Workflow Diagram",
    `<h1>Workflow Diagram</h1>${svg}`
  );
}

function generateSequence(data: any): string {
  const participants = data.participants ?? [];
  const messages = data.messages ?? [];
  const width = Math.max(600, participants.length * 160 + 80);
  const height = Math.max(300, messages.length * 60 + 150);

  let svg = `<svg width="${width}" height="${height}" xmlns="http://www.w3.org/2000/svg">`;
  svg += `<rect width="${width}" height="${height}" fill="white" rx="8"/>`;

  const colW = width / participants.length;
  const headerY = 40;
  const laneY = 80;

  // Participants header
  participants.forEach((p: any, i: number) => {
    const cx = colW * i + colW / 2;
    svg += `<rect x="${cx - 60}" y="${headerY}" width="120" height="32" fill="#f1f5f9" stroke="#cbd5e1" stroke-width="1" rx="6"/>`;
    svg += `<text x="${cx}" y="${headerY + 21}" font-size="12" fill="#334155" font-weight="600" text-anchor="middle">${esc(p.name || p.id)}</text>`;
    // Lifeline
    svg += `<line x1="${cx}" y1="${headerY + 32}" x2="${cx}" y2="${height - 20}" stroke="#e2e8f0" stroke-width="1" stroke-dasharray="4,4"/>`;
  });

  // Messages
  messages.forEach((msg: any, i: number) => {
    const fromIdx = participants.findIndex((p: any) => p.id === msg.from);
    const toIdx = participants.findIndex((p: any) => p.id === msg.to);
    if (fromIdx === -1 || toIdx === -1) return;

    const y = laneY + i * 50;
    const x1 = colW * fromIdx + colW / 2;
    const x2 = colW * toIdx + colW / 2;

    if (msg.type === "async") {
      svg += `<line x1="${x1}" y1="${y}" x2="${x2}" y2="${y}" stroke="#94a3b8" stroke-width="1" stroke-dasharray="6,3" marker-end="url(#arrow)"/>`;
    } else {
      const isSelf = msg.from === msg.to;
      if (isSelf) {
        svg += `<path d="M${x1},${y} C${x1 + 40},${y} ${x1 + 40},${y - 15} ${x1},${y - 15}" fill="none" stroke="#94a3b8" stroke-width="1.5" marker-end="url(#arrow)"/>`;
      } else {
        svg += `<line x1="${x1}" y1="${y}" x2="${x2}" y2="${y}" stroke="#94a3b8" stroke-width="1.5" marker-end="url(#arrow)"/>`;
      }
    }

    if (msg.label) {
      const mx = (x1 + x2) / 2;
      svg += `<text x="${mx}" y="${y - 6}" font-size="11" fill="#64748b" text-anchor="middle">${esc(msg.label)}</text>`;
    }
  });

  svg += `<defs><marker id="arrow" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L10,5 L0,10" fill="#94a3b8"/></marker></defs>`;

  svg += "</svg>";

  return baseHtml("Sequence Diagram",
    `<h1>Sequence Diagram</h1>${svg}`
  );
}

function generateDataflow(data: any): string {
  const sources = data.sources ?? [];
  const transforms = data.transforms ?? [];
  const sinks = data.sinks ?? [];
  const width = 900;
  const height = Math.max(300, (Math.max(sources.length, transforms.length, sinks.length)) * 100 + 100);

  let svg = `<svg width="${width}" height="${height}" xmlns="http://www.w3.org/2000/svg">`;
  svg += `<rect width="${width}" height="${height}" fill="white" rx="8"/>`;

  const cols = [
    { x: 50, label: "Sources", items: sources },
    { x: 340, label: "Transforms", items: transforms },
    { x: 630, label: "Sinks", items: sinks },
  ];

  cols.forEach((col) => {
    svg += `<text x="${col.x + 100}" y="32" font-size="14" fill="#64748b" font-weight="600" text-anchor="middle">${col.label}</text>`;

    col.items.forEach((item: any, i: number) => {
      const y = 60 + i * 90;
      const color = typeColor(item.type || "data");
      svg += `<rect x="${col.x}" y="${y}" width="200" height="50" fill="${color.bg}" stroke="${color.border}" stroke-width="1.5" rx="8"/>`;
      svg += `<text x="${col.x + 100}" y="${y + 30}" font-size="12" fill="${color.text}" font-weight="600" text-anchor="middle">${esc(item.label || item.id || `Item ${i + 1}`)}</text>`;
    });
  });

  // Draw connections between sources → transforms → sinks
  const maxRows = Math.max(sources.length, transforms.length, sinks.length);
  for (let i = 0; i < maxRows; i++) {
    if (i < sources.length) {
      const fromY = 60 + i * 90 + 25;
      if (i < transforms.length) {
        const toY = 60 + i * 90 + 25;
        svg += `<line x1="250" y1="${fromY}" x2="340" y2="${toY}" stroke="#94a3b8" stroke-width="1" marker-end="url(#arrow-h)"/>`;
      }
    }
    if (i < transforms.length) {
      const fromY = 60 + i * 90 + 25;
      if (i < sinks.length) {
        const toY = 60 + i * 90 + 25;
        svg += `<line x1="540" y1="${fromY}" x2="630" y2="${toY}" stroke="#94a3b8" stroke-width="1" marker-end="url(#arrow-h)"/>`;
      }
    }
  }

  svg += `<defs><marker id="arrow-h" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L10,5 L0,10" fill="#94a3b8"/></marker></defs>`;

  svg += "</svg>";

  return baseHtml("Dataflow Diagram",
    `<h1>Dataflow Diagram</h1>${svg}`
  );
}

function generateLifecycle(data: any): string {
  const states = data.states ?? [];
  const transitions = data.transitions ?? [];
  const width = 800;
  const height = 500;

  // Position states in a circle
  const cx = width / 2;
  const cy = height / 2;
  const radius = Math.min(width, height) / 2.5;

  states.forEach((state: any, i: number) => {
    const angle = (i / states.length) * 2 * Math.PI - Math.PI / 2;
    state.x = cx + radius * Math.cos(angle);
    state.y = cy + radius * Math.sin(angle);
    state.angle = angle;
  });

  let svg = `<svg width="${width}" height="${height}" xmlns="http://www.w3.org/2000/svg">`;
  svg += `<rect width="${width}" height="${height}" fill="white" rx="8"/>`;

  // Transitions
  transitions.forEach((t: any) => {
    const from = states.find((s: any) => s.id === t.from);
    const to = states.find((s: any) => s.id === t.to);
    if (!from || !to) return;

    const x1 = from.x, y1 = from.y;
    const x2 = to.x, y2 = to.y;

    if (t.from === t.to) {
      // Self-loop
      svg += `<path d="M${x1},${y1 - 25} C${x1 + 40},${y1 - 60} ${x1 + 40},${y1 + 10} ${x1},${y1 + 25}" fill="none" stroke="#94a3b8" stroke-width="1.5" marker-end="url(#arrow)"/>`;
      if (t.label) svg += `<text x="${x1 + 45}" y="${y1 - 25}" font-size="11" fill="#64748b">${esc(t.label)}</text>`;
    } else {
      const midX = (x1 + x2) / 2;
      const midY = (y1 + y2) / 2;
      svg += `<line x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}" stroke="#94a3b8" stroke-width="1.5" marker-end="url(#arrow)"/>`;
      if (t.label) svg += `<text x="${midX + 10}" y="${midY - 6}" font-size="11" fill="#64748b">${esc(t.label)}</text>`;
    }
  });

  // States
  states.forEach((state: any) => {
    const isInitial = state.type === "initial";
    const isFinal = state.type === "final";
    const fill = isInitial ? "#dbeafe" : isFinal ? "#fce7f3" : "#f1f5f9";
    const stroke = isInitial ? "#3b82f6" : isFinal ? "#ec4899" : "#cbd5e1";

    svg += `<ellipse cx="${state.x}" cy="${state.y}" rx="70" ry="28" fill="${fill}" stroke="${stroke}" stroke-width="2"/>`;
    svg += `<text x="${state.x}" y="${state.y + 4}" font-size="12" fill="#334155" font-weight="600" text-anchor="middle">${esc(state.label || state.id)}</text>`;
  });

  svg += `<defs><marker id="arrow" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L10,5 L0,10" fill="#94a3b8"/></marker></defs>`;

  svg += "</svg>";

  return baseHtml("Lifecycle Diagram",
    `<h1>Lifecycle Diagram</h1>${svg}`
  );
}

function typeColor(type: string): { bg: string; border: string; text: string } {
  const colors: Record<string, { bg: string; border: string; text: string }> = {
    service:     { bg: "#dbeafe", border: "#60a5fa", text: "#1e40af" },
    database:    { bg: "#e0e7ff", border: "#818cf8", text: "#3730a3" },
    cache:       { bg: "#fce7f3", border: "#f472b6", text: "#9d174d" },
    queue:       { bg: "#fef3c7", border: "#fbbf24", text: "#92400e" },
    api:         { bg: "#d1fae5", border: "#34d399", text: "#065f46" },
    client:      { bg: "#e0f2fe", border: "#38bdf8", text: "#075985" },
    storage:     { bg: "#f1f5f9", border: "#94a3b8", text: "#334155" },
    data:        { bg: "#f3e8ff", border: "#c084fc", text: "#6b21a8" },
    transform:   { bg: "#fff7ed", border: "#fb923c", text: "#7c2d12" },
    loadbalancer:{ bg: "#ecfdf5", border: "#6ee7b7", text: "#065f46" },
    gateway:     { bg: "#f5f3ff", border: "#a78bfa", text: "#5b21b6" },
    monitor:     { bg: "#fef2f2", border: "#f87171", text: "#991b1b" },
  };
  return colors[type.toLowerCase()] ?? { bg: "#f1f5f9", border: "#94a3b8", text: "#334155" };
}

function statusColor(status: string): { bg: string; border: string; text: string } {
  switch (status.toLowerCase()) {
    case "success": case "completed": return { bg: "#d1fae5", border: "#34d399", text: "#065f46" };
    case "running": case "in_progress": return { bg: "#dbeafe", border: "#60a5fa", text: "#1e40af" };
    case "failed": case "error": return { bg: "#fef2f2", border: "#f87171", text: "#991b1b" };
    case "pending": case "waiting": return { bg: "#fef3c7", border: "#fbbf24", text: "#92400e" };
    case "skipped": return { bg: "#f1f5f9", border: "#94a3b8", text: "#475569" };
    default: return { bg: "#f1f5f9", border: "#94a3b8", text: "#334155" };
  }
}

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}
