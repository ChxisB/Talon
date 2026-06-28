import { NextResponse } from "next/server";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import os from "node:os";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const CFG_DIR = path.join(os.homedir(), ".talon");

interface ActivityEntry {
  ts: number;
  type: string;
  agent: string;
  text: string;
}

export async function GET() {
  const entries: ActivityEntry[] = [];

  // ── Read tasks ──
  const tasksPath = path.join(CFG_DIR, "tasks.json");
  if (existsSync(tasksPath)) {
    try {
      const raw = JSON.parse(readFileSync(tasksPath, "utf8"));
      const tasks = raw.tasks || [];
      for (const t of tasks.slice(-30)) {
        let ts = t.updated_at ? new Date(t.updated_at).getTime() : Date.now();
        entries.push({ ts, type: "task", agent: "talon", text: `${t.status}: ${t.description}` });
        if (t.result) {
          entries.push({ ts: ts + 1, type: "result", agent: "talon", text: t.result.slice(0, 200) });
        }
      }
    } catch {}
  }

  // ── Read cron jobs ──
  const cronPath = path.join(CFG_DIR, "cron.json");
  if (existsSync(cronPath)) {
    try {
      const jobs = JSON.parse(readFileSync(cronPath, "utf8"));
      for (const j of jobs) {
        if (j.last_run) {
          entries.push({ ts: new Date(j.last_run).getTime(), type: "cron", agent: "talon", text: `Cron ran: ${j.name}` });
        }
      }
    } catch {}
  }

  // ── Sort by timestamp, newest first ──
  entries.sort((a, b) => b.ts - a.ts);

  return NextResponse.json({ entries: entries.slice(0, 80) });
}
