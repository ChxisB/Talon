import { NextResponse } from "next/server";
import { readFileSync, existsSync } from "node:fs";
import path from "node:path";
import os from "node:os";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

export async function GET() {
  const tasksPath = path.join(os.homedir(), ".talon", "tasks.json");
  
  if (!existsSync(tasksPath)) {
    return NextResponse.json({ tasks: [], stats: { total: 0, pending: 0, running: 0, completed: 0, failed: 0 } });
  }

  try {
    const raw = readFileSync(tasksPath, "utf8");
    const data = JSON.parse(raw);
    
    // Calculate stats
    const tasks = data.tasks || [];
    const stats: Record<string, number> = { total: tasks.length, pending: 0, running: 0, completed: 0, failed: 0, waiting: 0 };
    for (const t of tasks) {
      if (t.status in stats) stats[t.status]++;
    }

    return NextResponse.json({ tasks: tasks.slice(-50), stats });
  } catch {
    return NextResponse.json({ error: "Failed to parse tasks" }, { status: 500 });
  }
}
