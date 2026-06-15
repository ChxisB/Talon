import { NextRequest, NextResponse } from "next/server";
import { readFileSync, writeFileSync, existsSync } from "node:fs";
import path from "node:path";
import os from "node:os";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const CRON_PATH = path.join(os.homedir(), ".talon", "cron.json");

function readCron(): any[] {
  if (!existsSync(CRON_PATH)) return [];
  try {
    return JSON.parse(readFileSync(CRON_PATH, "utf8"));
  } catch {
    return [];
  }
}

function writeCron(jobs: any[]) {
  writeFileSync(CRON_PATH, JSON.stringify(jobs, null, 2));
}

export async function GET() {
  return NextResponse.json({ jobs: readCron() });
}

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const jobs = readCron();
    
    if (body.action === "add") {
      const job = {
        id: `cron_${Date.now()}`,
        name: body.name,
        prompt: body.prompt,
        interval: body.interval || "1d",
        enabled: true,
        created_at: new Date().toISOString(),
      };
      jobs.push(job);
      writeCron(jobs);
      return NextResponse.json({ status: "ok", job });
    }
    
    if (body.action === "remove" && body.id) {
      const filtered = jobs.filter((j: any) => j.id !== body.id);
      writeCron(filtered);
      return NextResponse.json({ status: "ok", removed: body.id });
    }
    
    if (body.action === "toggle" && body.id) {
      const idx = jobs.findIndex((j: any) => j.id === body.id);
      if (idx >= 0) {
        jobs[idx].enabled = !jobs[idx].enabled;
        writeCron(jobs);
        return NextResponse.json({ status: "ok", enabled: jobs[idx].enabled });
      }
    }
    
    return NextResponse.json({ error: "invalid action" }, { status: 400 });
  } catch {
    return NextResponse.json({ error: "invalid request" }, { status: 400 });
  }
}
