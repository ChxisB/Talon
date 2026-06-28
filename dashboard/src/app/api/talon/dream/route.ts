import { NextResponse } from "next/server";
import { readFileSync, readdirSync, existsSync } from "node:fs";
import path from "node:path";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const AGENT_URL = process.env.AGENT_INTERNAL_URL || "http://localhost:8082";
const VAULT_PATH = "/vault"; // mounted in Docker

export async function GET() {
  // Read vault notes
  if (!existsSync(VAULT_PATH)) {
    return NextResponse.json({ status: "no_vault", message: "Vault not mounted. Set VAULT_PATH." });
  }

  try {
    const notes: string[] = [];
    const walkDir = (dir: string, depth: number) => {
      if (depth > 5) return;
      let entries: { name: string; isDirectory(): boolean }[];
      try { entries = readdirSync(dir, { withFileTypes: true }); } catch { return; }
      for (const e of entries) {
        const fp = path.join(dir, e.name);
        if (e.isDirectory()) { if (!e.name.startsWith(".")) walkDir(fp, depth + 1); }
        else if (e.name.endsWith(".md")) { notes.push(fp); }
      }
    };
    walkDir(VAULT_PATH, 0);

    if (notes.length === 0) {
      return NextResponse.json({ status: "no_notes", message: "No markdown notes found in vault." });
    }

    const noteList = notes.slice(0, 20).map((n) => path.relative(VAULT_PATH, n)).join("\n");

    const prompt = `I have an Obsidian vault with ${notes.length} notes. Here are some of them:
${noteList}

Review these notes and suggest:
1. Connections between notes I may have missed ([[wikilinks]] to add)
2. A new insight or summary based on the content
3. A task to improve the vault organization`;

    const res = await fetch(`${AGENT_URL}/v1/messages`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        model: "",
        messages: [{ role: "user", content: [{ type: "text", text: prompt }] }],
        max_tokens: 4096,
        stream: false,
      }),
      signal: AbortSignal.timeout(60000),
    });

    if (!res.ok) {
      const errBody = await res.text().catch(() => "");
      return NextResponse.json({ status: "error", message: `Agent error: ${errBody}` });
    }

    // Parse SSE to get full text
    const reader = res.body?.getReader();
    if (!reader) return NextResponse.json({ status: "error", message: "No response body" });

    const decoder = new TextDecoder();
    let buffer = "";
    let fullText = "";
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() || "";
      for (const line of lines) {
        if (line.startsWith("data: ")) {
          try {
            const data = JSON.parse(line.slice(6));
            if (data.type === "content_block_delta" && data.delta?.type === "text_delta") {
              fullText += data.delta.text;
            }
          } catch {}
        }
      }
    }

    return NextResponse.json({
      status: "ok",
      notes_scanned: notes.length,
      insight: fullText,
    });
  } catch (err: any) {
    return NextResponse.json({ status: "error", message: err.message });
  }
}
