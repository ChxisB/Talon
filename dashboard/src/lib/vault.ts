import { readFile, readdir, stat } from "node:fs/promises";
import path from "node:path";

/**
 * VAULT_PATH is set via environment variable.
 * In Docker: VAULT_PATH=/vault (mounted from host ~/Talon/agent-vault)
 * In dev:    defaults to ~/Talon/agent-vault
 */
function resolveVaultRoot(): string {
  const env = process.env.VAULT_PATH;
  if (env) return env;
  const home = process.env.HOME || process.env.USERPROFILE || "/root";
  return path.join(home, "Talon", "agent-vault");
}

export const VAULT_ROOT = resolveVaultRoot();

const SKIP_DIRS = new Set([".obsidian", ".trash", "node_modules", ".git"]);

export function safeJoin(rel: string): string | null {
  const abs = path.resolve(VAULT_ROOT, rel);
  if (!abs.startsWith(VAULT_ROOT)) return null;
  return abs;
}

export async function listNotes(maxDepth = 6): Promise<string[]> {
  const out: string[] = [];
  async function walk(dir: string, depth: number) {
    if (depth > maxDepth) return;
    let items: { name: string; isDirectory(): boolean; isFile(): boolean }[];
    try { items = await readdir(dir, { withFileTypes: true }); }
    catch { return; }
    for (const it of items) {
      if (SKIP_DIRS.has(it.name)) continue;
      const full = path.join(dir, it.name);
      if (it.isDirectory()) {
        await walk(full, depth + 1);
      } else if (it.isFile() && /\.md$/i.test(it.name)) {
        out.push(full);
      }
    }
  }
  await walk(VAULT_ROOT, 0);
  return out;
}

export async function readNote(rel: string): Promise<{ path: string; content: string; mtime: number } | null> {
  const abs = safeJoin(rel);
  if (!abs) return null;
  if (!/\.md$/i.test(abs)) return null;
  try {
    const [content, st] = await Promise.all([readFile(abs, "utf8"), stat(abs)]);
    return { path: rel, content, mtime: st.mtimeMs };
  } catch { return null; }
}
