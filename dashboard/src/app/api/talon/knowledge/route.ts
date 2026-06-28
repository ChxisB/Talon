import { NextRequest, NextResponse } from "next/server";
import { existsSync } from "node:fs";
import { readFile, stat } from "node:fs/promises";
import path from "node:path";
import os from "node:os";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const AGENT_URL = process.env.AGENT_INTERNAL_URL || "http://localhost:8082";
const GRAPH_HTML = path.join(os.homedir(), ".talon", "knowledge", "graph.html");

/**
 * GET /api/talon/knowledge
 * Returns the current graph visualization HTML if it exists.
 */
export async function GET() {
  try {
    if (existsSync(GRAPH_HTML)) {
      const html = await readFile(GRAPH_HTML, "utf-8");
      const stats = await stat(GRAPH_HTML);
      return NextResponse.json({
        exists: true,
        html,
        lastBuilt: stats.mtime.toISOString(),
      });
    }
    return NextResponse.json({ exists: false });
  } catch (err: any) {
    return NextResponse.json({ error: err.message }, { status: 500 });
  }
}

/**
 * POST /api/talon/knowledge
 * Body: { action: "build"|"query"|"visualize", ... }
 *
 * Proxies to the Go server's /v1/graph/build endpoint instead of
 * spawning a subprocess (which failed because the binary wasn't found).
 */
export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    const { action } = body;

    if (!action) {
      return NextResponse.json({ error: "Missing action field" }, { status: 400 });
    }

    switch (action) {
      case "build": {
        const projectPath = body.path || ".";
        const res = await fetch(`${AGENT_URL}/v1/graph/build`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ path: projectPath }),
          signal: AbortSignal.timeout(120000), // 2 min timeout for large codebases
        });

        if (!res.ok) {
          const err = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
          return NextResponse.json({ error: err.error || "Build failed" }, { status: 500 });
        }

        const data = await res.json();

        // Cache the HTML for GET requests
        if (data.html) {
          const fs = await import("node:fs/promises");
          const dir = path.dirname(GRAPH_HTML);
          if (!existsSync(dir)) {
            await fs.mkdir(dir, { recursive: true });
          }
          await fs.writeFile(GRAPH_HTML, data.html, "utf-8");
        }

        return NextResponse.json({
          result: `Graph built: ${data.stats?.total_nodes || 0} nodes, ${data.stats?.total_edges || 0} edges`,
          stats: data.stats,
          html: data.html,
        });
      }

      case "query": {
        // For queries, we parse the cached graph data
        // In a full implementation, this would call a dedicated query endpoint
        return NextResponse.json({
          result: "Query endpoint via Go server not yet available. Build the graph first.",
        });
      }

      case "visualize": {
        if (existsSync(GRAPH_HTML)) {
          const html = await readFile(GRAPH_HTML, "utf-8");
          return NextResponse.json({ result: "Visualization loaded", html, path: GRAPH_HTML });
        }
        return NextResponse.json({ error: "No graph built yet" }, { status: 404 });
      }

      default:
        return NextResponse.json({ error: `Unknown action: ${action}` }, { status: 400 });
    }
  } catch (err: any) {
    return NextResponse.json({ error: err.message || String(err) }, { status: 500 });
  }
}
