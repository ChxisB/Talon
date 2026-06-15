import { NextResponse } from "next/server";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

/**
 * Proxies the agent status endpoint so the browser can fetch
 * it without CORS issues.
 *
 * Inside Docker: AGENT_INTERNAL_URL=http://agent:8082
 * Local dev:    defaults to http://localhost:8082
 */
const AGENT_URL = process.env.AGENT_INTERNAL_URL || "http://localhost:8082";

export async function GET() {
  const start = Date.now();

  try {
    // Hit the actual server endpoints instead of root (which returns 404)
    const [healthRes, versionRes] = await Promise.all([
      fetch(`${AGENT_URL}/v1/health`, { signal: AbortSignal.timeout(5000) }),
      fetch(`${AGENT_URL}/v1/version`, { signal: AbortSignal.timeout(5000) }).catch(() => null),
    ]);

    const latency = Date.now() - start;
    const healthOk = healthRes.ok;

    let version: string | null = null;
    let goVersion: string | null = null;
    let platform: string | null = null;
    if (versionRes && versionRes.ok) {
      try {
        const v = await versionRes.json();
        version = v.version ?? null;
        goVersion = v.go_version ?? null;
        platform = v.platform ?? null;
      } catch {}
    }

    return NextResponse.json({
      status: healthOk ? "ok" : "error",
      model: null,
      provider: null,
      latency,
      version,
      goVersion,
      platform,
    });
  } catch (err) {
    return NextResponse.json(
      {
        status: "error",
        model: null,
        provider: null,
        latency: null,
        error: err instanceof Error ? err.message : "Connection failed",
      },
      { status: 200 }
    );
  }
}
