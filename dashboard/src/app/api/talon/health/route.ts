import { NextResponse } from "next/server";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const AGENT_URL = process.env.AGENT_INTERNAL_URL || "http://localhost:8082";

export async function GET() {
  try {
    const res = await fetch(`${AGENT_URL}/v1/health`, {
      signal: AbortSignal.timeout(5000),
    });
    return NextResponse.json({ status: res.ok ? "healthy" : "unhealthy" });
  } catch {
    return NextResponse.json({ status: "unreachable" });
  }
}
