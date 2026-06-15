import { NextRequest, NextResponse } from "next/server";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const CACHE_SERVER = process.env.CACHE_URL || "http://localhost:8083";

/**
 * GET /api/talon/cache?action=list
 * GET /api/talon/cache?key=<key>
 */
export async function GET(req: NextRequest) {
  const { searchParams } = new URL(req.url);

  try {
    // List all keys (health check)
    const key = searchParams.get("key");
    if (!key) {
      try {
        const res = await fetch(`${CACHE_SERVER}/health`, { signal: AbortSignal.timeout(3000) });
        if (!res.ok) throw new Error("Cache server unhealthy");
        const health = await res.json();
        return NextResponse.json({ status: "ok", server: health });
      } catch {
        return NextResponse.json({ status: "unavailable", server: null });
      }
    }

    const res = await fetch(`${CACHE_SERVER}/get/${encodeURIComponent(key)}`, { signal: AbortSignal.timeout(3000) });
    if (res.status === 404) {
      return NextResponse.json({ error: "not found" }, { status: 404 });
    }
    if (!res.ok) throw new Error(`Cache error: ${res.status}`);
    const data = await res.json();
    return NextResponse.json(data);
  } catch (err) {
    const msg = err instanceof Error ? err.message : "Cache lookup failed";
    return NextResponse.json({ error: msg }, { status: 500 });
  }
}

/**
 * POST /api/talon/cache - set a cache entry
 * Body: { key, value, ttl? }
 */
export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    if (!body.key || body.value === undefined) {
      return NextResponse.json({ error: "key and value required" }, { status: 400 });
    }

    const res = await fetch(`${CACHE_SERVER}/set/${encodeURIComponent(body.key)}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ value: body.value, ttl: body.ttl || "24h" }),
      signal: AbortSignal.timeout(3000),
    });
    if (!res.ok) throw new Error(`Cache error: ${res.status}`);
    const data = await res.json();
    return NextResponse.json(data);
  } catch (err) {
    const msg = err instanceof Error ? err.message : "Cache set failed";
    return NextResponse.json({ error: msg }, { status: 500 });
  }
}

/**
 * DELETE /api/talon/cache?key=<key>
 */
export async function DELETE(req: NextRequest) {
  const { searchParams } = new URL(req.url);
  const key = searchParams.get("key");
  if (!key) {
    return NextResponse.json({ error: "key required" }, { status: 400 });
  }

  try {
    const res = await fetch(`${CACHE_SERVER}/del/${encodeURIComponent(key)}`, {
      method: "DELETE",
      signal: AbortSignal.timeout(3000),
    });
    if (!res.ok) throw new Error(`Cache error: ${res.status}`);
    const data = await res.json();
    return NextResponse.json(data);
  } catch (err) {
    const msg = err instanceof Error ? err.message : "Cache delete failed";
    return NextResponse.json({ error: msg }, { status: 500 });
  }
}
