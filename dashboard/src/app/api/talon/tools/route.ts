import { NextRequest, NextResponse } from "next/server";
import { readFileSync, writeFileSync, existsSync, mkdirSync } from "node:fs";
import path from "node:path";
import os from "node:os";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const TOOLS_CONFIG_PATH = path.join(os.homedir(), ".talon", "tools.json");

interface ToolConfig {
  enabled: boolean;
  level?: string;
}

interface ToolsState {
  tools: Record<string, ToolConfig>;
}

// Default configuration — all visible tools enabled
const DEFAULT_CONFIG: ToolsState = {
  tools: {
    reducer:      { enabled: true, level: "recommended" },
    viz:          { enabled: true },
    usage:        { enabled: true },
    graph:        { enabled: true },
  },
};

const TOOL_DEFINITIONS = [
  {
    id: "reducer",
    name: "Token Reducer",
    description: "Single master toggle for all token reduction: compresses tool outputs, LLM responses, command output, injects efficient coding principles, and caches results. Configure aggressiveness with the level setting.",
    icon: "compass",
    color: "#22c55e",
    configurable: true,
    levels: [
      { value: "recommended", label: "Recommended", desc: "Balanced savings (default) — 20 items kept, full compression, 150t minimum" },
      { value: "light", label: "Light", desc: "Minimal — 30 items kept, lite compression, 300t minimum" },
      { value: "moderate", label: "Moderate", desc: "Good — 15 items kept, full compression, 100t minimum" },
      { value: "aggressive", label: "Aggressive", desc: "Maximum — 10 items kept, ultra compression, 50t minimum, system prompt compressed" },
    ],
  },
  {
    id: "viz",
    name: "Viz (Diagram Generator)",
    description: "Generates self-contained HTML diagrams from JSON IR",
    icon: "layout",
    color: "#3b82f6",
    configurable: false,
  },
  {
    id: "usage",
    name: "Usage (Token Analytics)",
    description: "Parses session logs and generates token usage reports",
    icon: "bar-chart-2",
    color: "#06b6d4",
    configurable: false,
  },
  {
    id: "graph",
    name: "Graph (Knowledge Graph)",
    description: "Extracts code structure and builds queryable knowledge graphs",
    icon: "share-2",
    color: "#a855f7",
    configurable: false,
  },
];

function loadConfig(): ToolsState {
  try {
    if (existsSync(TOOLS_CONFIG_PATH)) {
      const data = readFileSync(TOOLS_CONFIG_PATH, "utf-8");
      const parsed = JSON.parse(data);
      // Merge with defaults to fill missing keys
      const merged = { ...DEFAULT_CONFIG };
      if (parsed.tools) {
        for (const [key, value] of Object.entries(parsed.tools)) {
          merged.tools[key] = value as ToolConfig;
        }
      }
      return merged;
    }
  } catch {}
  return DEFAULT_CONFIG;
}

function saveConfig(state: ToolsState): void {
  const dir = path.dirname(TOOLS_CONFIG_PATH);
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }
  writeFileSync(TOOLS_CONFIG_PATH, JSON.stringify(state, null, 2), "utf-8");
}

/**
 * GET /api/talon/tools
 * Returns the current tools configuration and definitions.
 */
export async function GET() {
  const config = loadConfig();

  const tools = TOOL_DEFINITIONS.map((def) => ({
    ...def,
    enabled: config.tools[def.id]?.enabled ?? true,
    level: config.tools[def.id]?.level ?? undefined,
  }));

  return NextResponse.json({ tools });
}

/**
 * PUT /api/talon/tools
 * Body: { id: string, enabled: boolean, level?: string }
 * Updates a single tool's configuration.
 */
export async function PUT(req: NextRequest) {
  try {
    const body = await req.json();
    const { id, enabled, level } = body;

    if (!id) {
      return NextResponse.json({ error: "Missing tool id" }, { status: 400 });
    }

    const def = TOOL_DEFINITIONS.find((d) => d.id === id);
    if (!def) {
      return NextResponse.json({ error: `Unknown tool: ${id}` }, { status: 404 });
    }

    const config = loadConfig();

    if (!config.tools[id]) {
      config.tools[id] = { enabled: true };
    }

    if (typeof enabled === "boolean") {
      config.tools[id].enabled = enabled;
    }

    if (level !== undefined && def.configurable) {
      config.tools[id].level = level;
    }

    saveConfig(config);

    return NextResponse.json({
      ok: true,
      tool: {
        id,
        enabled: config.tools[id].enabled,
        level: config.tools[id].level,
      },
    });
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? err.message : "Update failed" },
      { status: 500 }
    );
  }
}

/**
 * POST /api/talon/tools
 * Body: { action: "reset" }
 * Resets all tools to default configuration.
 */
export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const { action } = body;

    if (action === "reset") {
      saveConfig(DEFAULT_CONFIG);
      return NextResponse.json({ ok: true, tools: DEFAULT_CONFIG.tools });
    }

    return NextResponse.json({ error: `Unknown action: ${action}` }, { status: 400 });
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? err.message : "Action failed" },
      { status: 500 }
    );
  }
}
