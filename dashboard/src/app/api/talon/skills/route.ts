import { NextRequest, NextResponse } from "next/server";
import { readFileSync, writeFileSync, existsSync, mkdirSync } from "node:fs";
import path from "node:path";
import os from "node:os";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const SKILLS_CONFIG_PATH = path.join(os.homedir(), ".talon", "skills.json");

interface SkillToggle {
  enabled: boolean;
}

interface SkillsState {
  skills: Record<string, SkillToggle>;
}

// Known skills from discovery — loaded from disk if available, otherwise defaults
const KNOWN_SKILLS = [
  // Builtin
  { id: "jq", name: "jq", description: "JSON querying and reshaping", source: "builtin", builtin: true },
  { id: "talon-config", name: "Talon Config", description: "Talon configuration management", source: "builtin", builtin: true },
  { id: "talon-hooks", name: "Talon Hooks", description: "Author hook scripts to gate/rewrite tool calls", source: "builtin", builtin: true },
  // User skills
  { id: "talon-viz", name: "Talon Viz", description: "Generate architecture, workflow, and sequence diagrams", source: "user", builtin: false },
  { id: "ralph-tui-prd", name: "Ralph TUI PRD", description: "Generate Product Requirements Documents for ralph-tui", source: "user", builtin: false },
  { id: "ralph-tui-create-beads", name: "Create Beads", description: "Convert PRDs to beads format for ralph-tui", source: "user", builtin: false },
  { id: "ralph-tui-create-beads-rust", name: "Create Beads (Rust)", description: "Convert PRDs to beads using beads-rust CLI", source: "user", builtin: false },
  { id: "ralph-tui-create-json", name: "Create JSON Tasks", description: "Convert PRDs to prd.json for ralph-tui", source: "user", builtin: false },
];

const DEFAULT_CONFIG: SkillsState = {
  skills: Object.fromEntries(KNOWN_SKILLS.map(s => [s.id, { enabled: true }])),
};

function loadConfig(): SkillsState {
  try {
    if (existsSync(SKILLS_CONFIG_PATH)) {
      const data = readFileSync(SKILLS_CONFIG_PATH, "utf-8");
      const parsed = JSON.parse(data);
      const merged = { ...DEFAULT_CONFIG };
      if (parsed.skills) {
        for (const [key, value] of Object.entries(parsed.skills)) {
          merged.skills[key] = value as SkillToggle;
        }
      }
      return merged;
    }
  } catch {}
  return DEFAULT_CONFIG;
}

function saveConfig(state: SkillsState): void {
  const dir = path.dirname(SKILLS_CONFIG_PATH);
  if (!existsSync(dir)) mkdirSync(dir, { recursive: true });
  writeFileSync(SKILLS_CONFIG_PATH, JSON.stringify(state, null, 2), "utf-8");
}

/**
 * GET /api/talon/skills
 * Returns all known skills with their enabled/disabled state.
 */
export async function GET() {
  const config = loadConfig();

  const skills = KNOWN_SKILLS.map((def) => ({
    ...def,
    enabled: config.skills[def.id]?.enabled ?? true,
  }));

  return NextResponse.json({ skills });
}

/**
 * PUT /api/talon/skills
 * Body: { id: string, enabled: boolean }
 * Toggles a skill on/off.
 */
export async function PUT(req: NextRequest) {
  try {
    const body = await req.json();
    const { id, enabled } = body;

    if (!id) {
      return NextResponse.json({ error: "Missing skill id" }, { status: 400 });
    }

    const def = KNOWN_SKILLS.find((d) => d.id === id);
    if (!def) {
      return NextResponse.json({ error: `Unknown skill: ${id}` }, { status: 404 });
    }

    const config = loadConfig();

    if (!config.skills[id]) {
      config.skills[id] = { enabled: true };
    }

    if (typeof enabled === "boolean") {
      config.skills[id].enabled = enabled;
    }

    saveConfig(config);

    return NextResponse.json({ ok: true, skill: { id, enabled: config.skills[id].enabled } });
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? err.message : "Update failed" },
      { status: 500 }
    );
  }
}

/**
 * POST /api/talon/skills
 * Body: { action: "reset" }
 * Resets all skills to default (enabled).
 */
export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const { action } = body;

    if (action === "reset") {
      saveConfig(DEFAULT_CONFIG);
      return NextResponse.json({ ok: true, skills: DEFAULT_CONFIG.skills });
    }

    return NextResponse.json({ error: `Unknown action: ${action}` }, { status: 400 });
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? err.message : "Action failed" },
      { status: 500 }
    );
  }
}
