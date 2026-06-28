import { NextRequest, NextResponse } from "next/server";
import { readFile, writeFile, mkdir } from "node:fs/promises";
import { existsSync } from "node:fs";
import path from "node:path";
import os from "node:os";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const HOME = os.homedir();
const ENV_FILE = path.join(HOME, ".talon", ".env");
const ENV_DIR = path.join(HOME, ".talon");

// ─── Migration ─────────────────────────────────────────────────────

const LEGACY_ENV_FILE = path.join(HOME, ".fcc", ".env");

async function migrateLegacyEnv(): Promise<void> {
  if (existsSync(ENV_FILE) || !existsSync(LEGACY_ENV_FILE)) return;
  try {
    if (!existsSync(ENV_DIR)) await mkdir(ENV_DIR, { recursive: true });
    const data = await readFile(LEGACY_ENV_FILE, "utf8");
    await writeFile(ENV_FILE, data, "utf8");
    console.log("[talon] migrated config from ~/.fcc/.env → ~/.talon/.env");
  } catch { /* best-effort */ }
}

// ─── Env file helpers ───────────────────────────────────────────────

async function readEnvFile(): Promise<Record<string, string>> {
  try {
    const txt = await readFile(ENV_FILE, "utf8");
    const vals: Record<string, string> = {};
    for (const line of txt.split(/\r?\n/)) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith("#") || trimmed.startsWith("export ")) continue;
      const eq = trimmed.indexOf("=");
      if (eq === -1) continue;
      const k = trimmed.slice(0, eq).trim();
      let v = trimmed.slice(eq + 1).trim();
      if ((v.startsWith('"') && v.endsWith('"')) || (v.startsWith("'") && v.endsWith("'"))) {
        v = v.slice(1, -1);
      }
      vals[k] = v;
    }
    return vals;
  } catch {
    return {};
  }
}

async function writeEnvFile(updates: Record<string, string>): Promise<void> {
  const existing = await readEnvFile();
  const merged = { ...existing, ...updates };
  const sorted = Object.entries(merged).sort(([a], [b]) => {
    const aIsKey = a.endsWith("_API_KEY");
    const bIsKey = b.endsWith("_API_KEY");
    if (aIsKey && !bIsKey) return -1;
    if (!aIsKey && bIsKey) return 1;
    return a.localeCompare(b);
  });
  const lines = sorted.map(([k, v]) => {
    if (v === "" || v === null || v === undefined) {
      return `# ${k}=\n${k}=`;
    }
    return `${k}=${v}`;
  });
  if (!existsSync(ENV_DIR)) {
    await mkdir(ENV_DIR, { recursive: true });
  }
  await writeFile(ENV_FILE, lines.join("\n") + "\n", "utf8");
}

function maskSensitive(key: string, value: string): string {
  const sensitive = key.endsWith("_API_KEY") || key.endsWith("_TOKEN") || key.endsWith("_SECRET") || key.includes("KEY");
  if (sensitive && value.length > 4) return "••••" + value.slice(-4);
  if (sensitive && value.length > 0) return "••••";
  return value;
}

// ─── Route handlers ─────────────────────────────────────────────────

async function handleGET() {
  await migrateLegacyEnv();
  const env = await readEnvFile();
  const safe: Record<string, string> = {};

  // First, populate from env file on disk
  for (const [k, v] of Object.entries(env)) {
    safe[k] = maskSensitive(k, v);
  }

  // Then overlay any env vars that aren't in the file or are empty in the file
  // This allows Docker compose env vars to show up in the dashboard settings
  for (const k of API_KEY_ENV_VARS) {
    const envVal = process.env[k];
    if (envVal && !safe[k]) {
      safe[k] = maskSensitive(k, envVal);
    }
  }

  safe["_env_file"] = ENV_FILE;
  return NextResponse.json({ config: safe });
}

const API_KEY_ENV_VARS = [
  "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GEMINI_API_KEY",
  "OPENROUTER_API_KEY", "OPENCODE_API_KEY", "DEEPSEEK_API_KEY",
  "MISTRAL_API_KEY", "GROQ_API_KEY", "CEREBRAS_API_KEY",
  "NVIDIA_NIM_API_KEY",
];

async function handlePUT(body: unknown) {
  if (typeof body !== "object" || body === null) {
    return NextResponse.json({ error: "invalid JSON body" }, { status: 400 });
  }
  const raw = body as Record<string, string>;
  // Read existing values so we can preserve keys the user didn't touch
  const existing = await readEnvFile();
  // Filter: skip masked values, skip empty API keys (preserve existing)
  const updates: Record<string, string> = {};
  for (const [k, v] of Object.entries(raw)) {
    if (v.startsWith("••••")) continue; // skip masked values
    if (v === "" && (k.endsWith("_API_KEY") || k.endsWith("_TOKEN") || k.endsWith("_SECRET") || k.includes("KEY"))) {
      if ((existing[k] ?? "") !== "") continue; // preserve existing non-empty key
    }
    updates[k] = v;
  }
  const keys = Object.keys(updates);
  if (keys.length === 0) {
    return NextResponse.json({ status: "ok", updated: 0, note: "no unmasked values to save" });
  }
  try {
    await writeEnvFile(updates);
    return NextResponse.json({ status: "ok", updated: keys.length });
  } catch (err) {
    return NextResponse.json(
      { error: `write failed: ${err instanceof Error ? err.message : String(err)}` },
      { status: 500 }
    );
  }
}

async function handlePOST(body: unknown) {
  const req = (typeof body === "object" && body !== null ? body : {}) as Record<string, string>;
  const provider = req["provider"] || "";
  const apiKey = req["api_key"] || "";
  return NextResponse.json({
    provider,
    valid: apiKey.length > 0,
    message: apiKey ? "API key provided" : "No API key provided",
  });
}

// ─── Catch-all route ────────────────────────────────────────────────

export async function GET(req: NextRequest) {
  const route = new URL(req.url).pathname.replace(/^\/api\/talon\/admin\//, "");
  switch (route) {
    case "config": return handleGET();
    default: return NextResponse.json({ error: "not found" }, { status: 404 });
  }
}

export async function PUT(req: NextRequest) {
  const route = new URL(req.url).pathname.replace(/^\/api\/talon\/admin\//, "");
  switch (route) {
    case "config": {
      const body = await req.json().catch(() => null);
      return handlePUT(body);
    }
    default: return NextResponse.json({ error: "not found" }, { status: 404 });
  }
}

export async function POST(req: NextRequest) {
  const route = new URL(req.url).pathname.replace(/^\/api\/talon\/admin\//, "");
  switch (route) {
    case "validate": {
      const body = await req.json().catch(() => null);
      return handlePOST(body);
    }
    default: return NextResponse.json({ error: "not found" }, { status: 404 });
  }
}
