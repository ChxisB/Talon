import { NextRequest, NextResponse } from "next/server";
import { readFileSync, writeFileSync, readdirSync, existsSync, unlinkSync, mkdirSync } from "node:fs";
import path from "node:path";
import os from "node:os";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const AGENTS_DIR = path.join(os.homedir(), ".talon", "agents");

// ─── Default agents (seeded on first request if directory is empty) ──

const DEFAULT_AGENTS = [
  {
    name: "Code Reviewer",
    keywords: ["review", "code review", "typescript", "react"],
    instructions: `You are an expert code reviewer. Focus on:
- Type safety and edge cases
- Performance implications
- Readability and maintainability
- Potential bugs and anti-patterns
- Suggest concrete improvements`,
  },
  {
    name: "Architecture Advisor",
    keywords: ["architecture", "design", "system design"],
    instructions: `You are a software architecture advisor. Help with:
- System design and trade-offs
- Technology selection
- Scalability considerations
- Design patterns and their applicability
- API design principles`,
  },
  {
    name: "Test Writer",
    keywords: ["test", "testing", "unit test", "integration"],
    instructions: `You are a testing specialist. When writing tests:
- Follow the AAA pattern (Arrange, Act, Assert)
- Test edge cases and error states
- Mock external dependencies
- Keep tests isolated and deterministic
- Aim for high coverage on critical paths`,
  },
];

function parseAgentMD(content: string) {
  const lines = content.split("\n---\n");
  if (lines.length < 2) {
    const parts = content.trim().split("\n");
    return { name: parts[0].replace(/^#\s*/, "").trim(), keywords: [], instructions: parts.slice(1).join("\n").trim() };
  }
  const header = lines[0].trim();
  const body = lines.slice(1).join("\n---\n").trim();
  const name = header.replace(/^#\s*/, "").trim();
  const kwMatch = header.match(/Keywords:\s*(.+)/i);
  const keywords = kwMatch ? kwMatch[1].split(",").map((k: string) => k.trim()).filter(Boolean) : [];
  return { name, keywords, instructions: body };
}

function agentToMD(a: { name: string; keywords?: string[]; instructions: string }) {
  return `# ${a.name}
Keywords: ${(a.keywords || []).join(", ")}
---
${a.instructions}
`;
}

function fileName(name: string) {
  return name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "") + ".md";
}

function seedDefaultAgents() {
  if (!existsSync(AGENTS_DIR)) {
    mkdirSync(AGENTS_DIR, { recursive: true });
  }
  const existing = readdirSync(AGENTS_DIR).filter((f) => f.endsWith(".md"));
  if (existing.length > 0) return; // already has agents
  for (const agent of DEFAULT_AGENTS) {
    const fname = fileName(agent.name);
    const fp = path.join(AGENTS_DIR, fname);
    if (!existsSync(fp)) {
      writeFileSync(fp, agentToMD(agent));
    }
  }
}

export async function GET() {
  try {
    // Seed default agents on first access if empty
    seedDefaultAgents();

    const files = readdirSync(AGENTS_DIR).filter((f) => f.endsWith(".md"));
    const agents = files.map((f) => {
      const content = readFileSync(path.join(AGENTS_DIR, f), "utf8");
      return { ...parseAgentMD(content), file: f };
    });
    return NextResponse.json({ agents });
  } catch {
    return NextResponse.json({ agents: [] });
  }
}

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    if (!existsSync(AGENTS_DIR)) {
      require("node:fs").mkdirSync(AGENTS_DIR, { recursive: true });
    }

    if (body.action === "update" && body.file) {
      const fp = path.join(AGENTS_DIR, body.file);
      if (!existsSync(fp)) return NextResponse.json({ error: "not found" }, { status: 404 });
      writeFileSync(fp, agentToMD({ name: body.name, keywords: body.keywords, instructions: body.instructions }));
      return NextResponse.json({ status: "ok" });
    }

    if (body.action === "delete" && body.file) {
      const fp = path.join(AGENTS_DIR, body.file);
      if (!existsSync(fp)) return NextResponse.json({ error: "not found" }, { status: 404 });
      unlinkSync(fp);
      return NextResponse.json({ status: "ok", deleted: body.file });
    }

    if (body.action === "create") {
      const fname = fileName(body.name);
      const fp = path.join(AGENTS_DIR, fname);
      if (existsSync(fp)) return NextResponse.json({ error: "already exists" }, { status: 409 });
      writeFileSync(fp, agentToMD({ name: body.name, keywords: body.keywords || [], instructions: body.instructions }));
      return NextResponse.json({ status: "ok", file: fname });
    }

    return NextResponse.json({ error: "invalid action" }, { status: 400 });
  } catch {
    return NextResponse.json({ error: "invalid request" }, { status: 400 });
  }
}
