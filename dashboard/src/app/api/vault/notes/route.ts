import { NextRequest, NextResponse } from "next/server";
import { readFileSync, readdirSync, statSync, existsSync } from "node:fs";
import path from "node:path";
import os from "node:os";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

// Fallback vault paths for local dev
function findVault(): string {
  const envPath = process.env.VAULT_PATH;
  if (envPath && existsSync(envPath)) return envPath;
  const home = os.homedir();
  const candidates = [
    path.join(home, ".talon", "vault"),
    path.join(home, "Talon", "agent-vault"),
    path.join(home, "agent-vault"),
    "/vault",
  ];
  for (const c of candidates) {
    if (existsSync(c)) return c;
  }
  return envPath || "/vault";
}

const VAULT_PATH = findVault();

export async function GET(req: NextRequest) {
  const url = new URL(req.url);
  const notePath = url.searchParams.get("path");

  // Single note content
  if (notePath) {
    const fp = path.join(VAULT_PATH, notePath.replace(/^\/+/, ""));
    if (!existsSync(fp)) return NextResponse.json({ error: "not found" }, { status: 404 });
    try {
      const content = readFileSync(fp, "utf8");
      const st = statSync(fp);
      return NextResponse.json({ path: notePath, content, mtime: st.mtimeMs });
    } catch {
      return NextResponse.json({ error: "read failed" }, { status: 500 });
    }
  }

  // List all notes
  if (!existsSync(VAULT_PATH)) return NextResponse.json({ notes: [] });
  try {
    const notes: { path: string; title: string; group: string; mtime: number }[] = [];
    const walkDir = (dir: string, depth: number) => {
      if (depth > 5) return;
      let entries: { name: string; isDirectory(): boolean }[];
      try { entries = readdirSync(dir, { withFileTypes: true }); } catch { return; }
      for (const e of entries) {
        const fp = path.join(dir, e.name);
        if (e.isDirectory()) { if (!e.name.startsWith(".")) walkDir(fp, depth + 1); }
        else if (e.name.endsWith(".md")) {
          const rel = path.relative(VAULT_PATH, fp);
          const group = rel.includes(path.sep) ? rel.split(path.sep)[0] : "root";
          let mtime = 0;
          try { mtime = statSync(fp).mtimeMs; } catch {}
          notes.push({ path: rel, title: e.name.replace(/\.md$/, ""), group, mtime });
        }
      }
    };
    walkDir(VAULT_PATH, 0);
    notes.sort((a, b) => b.mtime - a.mtime);
    return NextResponse.json({ notes });
  } catch {
    return NextResponse.json({ notes: [] });
  }
}
