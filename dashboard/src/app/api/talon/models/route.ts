import { NextResponse } from "next/server";
import { readFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import path from "node:path";
import os from "node:os";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const ENV_FILE = path.join(os.homedir(), ".talon", ".env");
const AGENT_URL = process.env.AGENT_INTERNAL_URL || "http://localhost:8082";

interface ModelEntry {
  id: string;
  provider: string;
  subgroup?: string;
  name: string;
  cost_per_1m_in: number;
  cost_per_1m_out: number;
  context_window: number;
}

/**
 * GET /api/talon/models
 *
 * Priority:
 *   1. Fetch models dynamically from each configured provider's live API
 *   2. Fall back to Go server /v1/models for provider names
 *   3. Fall back to workspace providers endpoint
 */
export async function GET() {
  // 1. Fetch dynamically from provider APIs using configured API keys
  const liveModels = await fetchFromProviderAPIs();
  if (liveModels.length > 0) {
    return NextResponse.json({ data: liveModels });
  }

  // 2. Fallback: try Go server's /v1/models
  try {
    const res = await fetch(`${AGENT_URL}/v1/models`, { signal: AbortSignal.timeout(5000) });
    if (res.ok) {
      const data = await res.json();
      if (data?.data?.length > 0) return NextResponse.json(data);
    }
  } catch {}

  // 3. Fallback: workspace providers endpoint
  try {
    const res = await fetch(`${AGENT_URL}/v1/workspaces/default/providers`, { signal: AbortSignal.timeout(5000) });
    if (res.ok) {
      const providers = await res.json();
      if (Array.isArray(providers) && providers.length > 0) {
        const models: ModelEntry[] = [];
        for (const p of providers) {
          for (const m of (p.models || [])) {
            models.push({ id: `${p.id}/${m.id}`, provider: p.id, name: m.name || m.id, cost_per_1m_in: 0, cost_per_1m_out: 0, context_window: 128000 });
          }
        }
        if (models.length > 0) return NextResponse.json({ data: models });
      }
    }
  } catch {}

  return NextResponse.json({ data: [] });
}

async function fetchOpenAI(key: string): Promise<ModelEntry[]> {
  try {
    const r = await fetch("https://api.openai.com/v1/models", {
      headers: { Authorization: `Bearer ${key}` }, signal: AbortSignal.timeout(5000)
    });
    if (!r.ok) return [];
    const d = await r.json();
    return (d.data || []).map((m: any) => ({
      id: `openai/${m.id}`, provider: "openai", name: m.id,
      cost_per_1m_in: 0, cost_per_1m_out: 0, context_window: 128000,
    }));
  } catch { return []; }
}

async function fetchAnthropic(key: string): Promise<ModelEntry[]> {
  try {
    const r = await fetch("https://api.anthropic.com/v1/models", {
      headers: { "x-api-key": key, "anthropic-version": "2023-06-01" }, signal: AbortSignal.timeout(5000)
    });
    if (!r.ok) return [];
    const d = await r.json();
    return (d.data || []).map((m: any) => ({
      id: `anthropic/${m.id}`, provider: "anthropic", name: m.display_name || m.id,
      cost_per_1m_in: 0, cost_per_1m_out: 0, context_window: 200000,
    }));
  } catch { return []; }
}

async function fetchGemini(key: string): Promise<ModelEntry[]> {
  try {
    const r = await fetch("https://generativelanguage.googleapis.com/v1beta/models?key=" + key, {
      signal: AbortSignal.timeout(5000)
    });
    if (!r.ok) return [];
    const d = await r.json();
    return (d.models || []).filter((m: any) => m.name.startsWith("models/gemini-")).map((m: any) => ({
      id: `google/${m.name.replace("models/", "")}`, provider: "google", name: m.displayName || m.name,
      cost_per_1m_in: 0, cost_per_1m_out: 0, context_window: 1000000,
    }));
  } catch { return []; }
}

async function fetchOpenRouter(key: string): Promise<ModelEntry[]> {
  try {
    const r = await fetch("https://openrouter.ai/api/v1/models", {
      headers: { Authorization: `Bearer ${key}` }, signal: AbortSignal.timeout(5000)
    });
    if (!r.ok) return [];
    const d = await r.json();
    return (d.data || []).map((m: any) => ({
      id: `open_router/${m.id}`, provider: "open_router", name: m.name || m.id,
      cost_per_1m_in: m.pricing?.prompt || 0, cost_per_1m_out: m.pricing?.completion || 0,
      context_window: m.context_length || 128000,
    }));
  } catch { return []; }
}

async function fetchOpenCode(key: string): Promise<ModelEntry[]> {
  const models: ModelEntry[] = [];
  try {
    const r = await fetch("https://opencode.ai/zen/v1/models", {
      headers: { Authorization: `Bearer ${key}` }, signal: AbortSignal.timeout(5000)
    });
    if (r.ok) {
      const items = (await r.json()).data || [];
      for (const m of items) {
        models.push({ id: `opencode/${m.id}`, provider: "opencode", subgroup: "zen", name: m.id, cost_per_1m_in: 0, cost_per_1m_out: 0, context_window: 128000 });
      }
    }
  } catch {}
  try {
    const r = await fetch("https://opencode.ai/zen/go/v1/models", {
      headers: { Authorization: `Bearer ${key}` }, signal: AbortSignal.timeout(5000)
    });
    if (r.ok) {
      const items = (await r.json()).data || [];
      for (const m of items) {
        models.push({ id: `opencode_go/${m.id}`, provider: "opencode", subgroup: "go", name: m.id, cost_per_1m_in: 0, cost_per_1m_out: 0, context_window: 128000 });
      }
    }
  } catch {}
  return models;
}

async function fetchDeepSeek(key: string): Promise<ModelEntry[]> {
  try {
    const r = await fetch("https://api.deepseek.com/v1/models", {
      headers: { Authorization: `Bearer ${key}` }, signal: AbortSignal.timeout(5000)
    });
    if (!r.ok) return [];
    const d = await r.json();
    return (d.data || []).map((m: any) => ({
      id: `deepseek/${m.id}`, provider: "deepseek", name: m.id,
      cost_per_1m_in: 0, cost_per_1m_out: 0, context_window: 128000,
    }));
  } catch { return []; }
}

async function fetchFromProviderAPIs(): Promise<ModelEntry[]> {
  const envVars: Record<string, string> = {};
  try {
    if (existsSync(ENV_FILE)) {
      const txt = await readFile(ENV_FILE, "utf8");
      for (const line of txt.split(/\r?\n/)) {
        const trimmed = line.trim();
        if (!trimmed || trimmed.startsWith("#") || trimmed.startsWith("export ")) continue;
        const eq = trimmed.indexOf("=");
        if (eq === -1) continue;
        const k = trimmed.slice(0, eq).trim();
        let v = trimmed.slice(eq + 1).trim();
        if ((v.startsWith('"') && v.endsWith('"')) || (v.startsWith("'") && v.endsWith("'"))) v = v.slice(1, -1);
        envVars[k] = v;
      }
    }
  } catch {}

  for (const key of ["OPENAI_API_KEY","ANTHROPIC_API_KEY","GEMINI_API_KEY","OPENROUTER_API_KEY","OPENCODE_API_KEY","DEEPSEEK_API_KEY","MISTRAL_API_KEY","GROQ_API_KEY","CEREBRAS_API_KEY","NVIDIA_NIM_API_KEY"]) {
    if (process.env[key] && !envVars[key]) envVars[key] = process.env[key]!;
  }

  const results = await Promise.allSettled([
    envVars["OPENAI_API_KEY"]    ? fetchOpenAI(envVars["OPENAI_API_KEY"])    : Promise.resolve([] as ModelEntry[]),
    envVars["ANTHROPIC_API_KEY"] ? fetchAnthropic(envVars["ANTHROPIC_API_KEY"]) : Promise.resolve([] as ModelEntry[]),
    envVars["GEMINI_API_KEY"]    ? fetchGemini(envVars["GEMINI_API_KEY"])    : Promise.resolve([] as ModelEntry[]),
    envVars["OPENROUTER_API_KEY"]? fetchOpenRouter(envVars["OPENROUTER_API_KEY"]) : Promise.resolve([] as ModelEntry[]),
    envVars["OPENCODE_API_KEY"]  ? fetchOpenCode(envVars["OPENCODE_API_KEY"])  : Promise.resolve([] as ModelEntry[]),
    envVars["DEEPSEEK_API_KEY"]  ? fetchDeepSeek(envVars["DEEPSEEK_API_KEY"])  : Promise.resolve([] as ModelEntry[]),
  ]);

  const allModels: ModelEntry[] = [];
  for (const r of results) {
    if (r.status === "fulfilled") allModels.push(...r.value);
  }

  const seen = new Set<string>();
  return allModels.filter(m => { if (seen.has(m.id)) return false; seen.add(m.id); return true; })
    .sort((a, b) => a.provider.localeCompare(b.provider) || a.name.localeCompare(b.name));
}
