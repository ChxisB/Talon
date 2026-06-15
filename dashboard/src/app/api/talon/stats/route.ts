import { NextResponse } from "next/server";
import { execSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import os from "node:os";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

interface TotalStats {
  total_sessions: number;
  total_prompt_tokens: number;
  total_completion_tokens: number;
  total_cost: number;
  total_messages: number;
  avg_tokens_per_session: number;
  avg_messages_per_session: number;
}
interface DailyUsage { day: string; prompt_tokens: number; completion_tokens: number; cost: number; session_count: number; }
interface ModelUsage { model: string; provider: string; message_count: number; }

const AGENT_URL = process.env.AGENT_INTERNAL_URL || "http://localhost:8082";

/** Try fetching stats from the Go agent server (works in Docker). */
async function fetchFromAgent(): Promise<{ total: TotalStats; daily: DailyUsage[]; models: ModelUsage[] } | null> {
  try {
    // Get the first workspace ID
    const wsRes = await fetch(`${AGENT_URL}/v1/workspaces`, {
      signal: AbortSignal.timeout(3000),
    });
    if (!wsRes.ok) return null;
    const workspaces = await wsRes.json() as { id: string }[];
    if (!workspaces?.length) return null;

    // Fetch stats from the first workspace
    const statsRes = await fetch(`${AGENT_URL}/v1/workspaces/${workspaces[0].id}/stats`, {
      signal: AbortSignal.timeout(5000),
    });
    if (!statsRes.ok) return null;
    return statsRes.json();
  } catch {
    return null;
  }
}

/** Fallback: read stats directly from talon.db via sqlite3 CLI. */
function findDbPath(): string | null {
  if (process.env.TALON_DATA_DIR) {
    const p = path.join(process.env.TALON_DATA_DIR, "talon.db");
    if (existsSync(p)) return p;
  }

  let dir = process.cwd();
  for (let i = 0; i < 5; i++) {
    const candidate = path.join(dir, ".talon", "talon.db");
    if (existsSync(candidate)) return candidate;
    const parent = path.dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }

  const homeDb = path.join(os.homedir(), ".talon", "talon.db");
  if (existsSync(homeDb)) return homeDb;

  const projectsJson = path.join(
    process.env.XDG_DATA_HOME || path.join(os.homedir(), ".local", "share"),
    "talon",
    "projects.json"
  );
  if (existsSync(projectsJson)) {
    try {
      const data = JSON.parse(readFileSync(projectsJson, "utf8"));
      const cwd = process.cwd();
      for (const proj of data.projects || []) {
        if (cwd.startsWith(proj.path) || proj.path.startsWith(cwd)) {
          const p = path.join(proj.data_dir, "talon.db");
          if (existsSync(p)) return p;
        }
      }
      const first = data.projects?.[0];
      if (first) {
        const p = path.join(first.data_dir, "talon.db");
        if (existsSync(p)) return p;
      }
    } catch {}
  }

  return null;
}

function queryJson<T>(dbPath: string, query: string): T[] {
  const out = execSync(`sqlite3 -json "${dbPath}" "${query}"`, {
    encoding: "utf8",
    timeout: 5000,
  });
  return JSON.parse(out || "[]");
}

async function fetchFromLocalDb(): Promise<{ total: TotalStats; daily: DailyUsage[]; models: ModelUsage[] } | null> {
  const dbPath = findDbPath();
  if (!dbPath) return null;

  try {
    const [totalArr, daily, models] = await Promise.all([
      queryJson<TotalStats>(
        dbPath,
        `SELECT
           COUNT(*) as total_sessions,
           COALESCE(SUM(prompt_tokens), 0) as total_prompt_tokens,
           COALESCE(SUM(completion_tokens), 0) as total_completion_tokens,
           ROUND(COALESCE(SUM(cost), 0), 6) as total_cost,
           COALESCE(SUM(message_count), 0) as total_messages,
           COALESCE(AVG(prompt_tokens + completion_tokens), 0) as avg_tokens_per_session,
           COALESCE(AVG(message_count), 0) as avg_messages_per_session
         FROM sessions
         WHERE parent_session_id IS NULL`
      ),
      queryJson<DailyUsage>(
        dbPath,
        `SELECT
           date(created_at/1000, 'unixepoch') as day,
           SUM(prompt_tokens) as prompt_tokens,
           SUM(completion_tokens) as completion_tokens,
           ROUND(SUM(cost), 6) as cost,
           COUNT(*) as session_count
         FROM sessions
         WHERE parent_session_id IS NULL
         GROUP BY day
         ORDER BY day DESC
         LIMIT 30`
      ),
      queryJson<ModelUsage>(
        dbPath,
        `SELECT
           COALESCE(model, 'unknown') as model,
           COALESCE(provider, 'unknown') as provider,
           COUNT(*) as message_count
         FROM messages
         WHERE role = 'assistant'
         GROUP BY model, provider
         ORDER BY message_count DESC`
      ),
    ]);

    const total = totalArr[0] || {
      total_sessions: 0, total_prompt_tokens: 0, total_completion_tokens: 0,
      total_cost: 0, total_messages: 0, avg_tokens_per_session: 0, avg_messages_per_session: 0,
    };

    return { total, daily, models };
  } catch {
    return null;
  }
}

export async function GET() {
  // Try agent server first (Docker/remote), fall back to local DB
  const data = (await fetchFromAgent()) || (await fetchFromLocalDb());

  if (!data) {
    return NextResponse.json(
      { error: "Stats unavailable. Ensure the agent is running or talon.db is accessible." },
      { status: 404 }
    );
  }

  return NextResponse.json(data);
}
