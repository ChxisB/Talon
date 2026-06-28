import { readFile, stat } from "node:fs/promises";
import path from "node:path";
import { listNotes, VAULT_ROOT } from "./vault";

export interface GraphNode {
  id: string;
  title: string;
  group: string;
  degree: number;
  mtime: number;
}
export interface GraphLink { source: string; target: string; }
export interface VaultGraph { nodes: GraphNode[]; links: GraphLink[]; }

const WIKILINK_RE = /\[\[([^\[\]\n|#]+)(?:#[^\[\]\n|]+)?(?:\|[^\[\]\n]+)?\]\]/g;

export async function buildVaultGraph(): Promise<VaultGraph> {
  const files = await listNotes();

  const byTitle = new Map<string, string>();
  const meta = new Map<string, { rel: string; title: string; group: string; mtime: number }>();

  await Promise.all(files.map(async (abs) => {
    const rel = path.relative(VAULT_ROOT, abs);
    const title = path.basename(abs, ".md");
    const head = rel.split(path.sep)[0] || "root";
    const group = rel.includes(path.sep) ? head : "root";
    let mtime = 0;
    try { const st = await stat(abs); mtime = st.mtimeMs; } catch {}
    meta.set(rel, { rel, title, group, mtime });
    if (!byTitle.has(title.toLowerCase())) byTitle.set(title.toLowerCase(), rel);
  }));

  const linkSet = new Set<string>();
  const links: GraphLink[] = [];
  const degree = new Map<string, number>();

  await Promise.all(files.map(async (abs) => {
    const rel = path.relative(VAULT_ROOT, abs);
    let content = "";
    try { content = await readFile(abs, "utf8"); } catch { return; }
    const matches = content.matchAll(WIKILINK_RE);
    for (const m of matches) {
      const targetTitle = m[1].trim();
      if (!targetTitle) continue;
      const targetRel = byTitle.get(targetTitle.toLowerCase());
      if (!targetRel || targetRel === rel) continue;
      const key = rel + "→" + targetRel;
      if (linkSet.has(key)) continue;
      linkSet.add(key);
      links.push({ source: rel, target: targetRel });
      degree.set(rel, (degree.get(rel) ?? 0) + 1);
      degree.set(targetRel, (degree.get(targetRel) ?? 0) + 1);
    }
  }));

  const nodes: GraphNode[] = Array.from(meta.values()).map((m) => ({
    id: m.rel,
    title: m.title,
    group: m.group,
    degree: degree.get(m.rel) ?? 0,
    mtime: m.mtime,
  }));

  return { nodes, links };
}
