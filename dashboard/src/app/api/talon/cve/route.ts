import { NextRequest, NextResponse } from "next/server";
import { readFileSync, writeFileSync, existsSync, mkdirSync } from "node:fs";
import path from "node:path";
import os from "node:os";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const CVE_CACHE_DIR = path.join(os.homedir(), ".talon", "cve-cache");
const CVE_CACHE_FILE = path.join(CVE_CACHE_DIR, "cve-data.json");
const CVE_DATA_URL = "https://kazepublic.blob.core.windows.net/cvefree/data.json";
const CVE_META_FILE = path.join(CVE_CACHE_DIR, "meta.json");

interface CVE {
  cve: string;
  last_modified_datetime: string;
  published_datetime: string;
  description: string;
  vendors: string[];
  cvssv2: number | null;
  cvssv3: number | null;
  epss: number | null;
  v_score: number | null;
  cti_count: number;
  social_media_audience: number;
  software_cpes?: string[];
  cisa: boolean;
  metasploit: boolean;
}

interface CVEMeta {
  last_updated: string;
  total: number;
}

interface SearchQuery {
  search?: string;
  vendor?: string;
  severity?: string;
  cisa?: boolean;
  metasploit?: boolean;
  min_cvss?: number;
  max_cvss?: number;
  sort_by?: string;
  sort_dir?: string;
  page?: number;
  size?: number;
}

function severity(cve: CVE): string {
  const s = cve.cvssv3 ?? cve.cvssv2;
  if (s == null) return "unknown";
  if (s >= 9.0) return "critical";
  if (s >= 7.0) return "high";
  if (s >= 4.0) return "medium";
  if (s > 0) return "low";
  return "unknown";
}

function bestCVSS(cve: CVE): number {
  return cve.cvssv3 ?? cve.cvssv2 ?? 0;
}

let cachedCVEs: CVE[] | null = null;

function loadCache(): CVE[] {
  if (cachedCVEs) return cachedCVEs;
  try {
    if (existsSync(CVE_CACHE_FILE)) {
      const data = readFileSync(CVE_CACHE_FILE, "utf-8");
      const parsed = JSON.parse(data);
      // Handle both plain array and { metadata, cves } formats
      if (Array.isArray(parsed)) {
        cachedCVEs = parsed;
      } else if (parsed.cves && Array.isArray(parsed.cves)) {
        cachedCVEs = parsed.cves;
      }
      return cachedCVEs!;
    }
  } catch {}
  return [];
}

async function downloadData(): Promise<CVE[]> {
  console.log(`Downloading CVE data from ${CVE_DATA_URL}...`);
  const res = await fetch(CVE_DATA_URL);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  const parsed = await res.json();

  // Handle both plain array and { metadata, cves } formats
  let cves: CVE[];
  if (Array.isArray(parsed)) {
    cves = parsed;
  } else if (parsed.cves && Array.isArray(parsed.cves)) {
    cves = parsed.cves;
  } else {
    throw new Error("Unexpected CVE data format");
  }

  if (!existsSync(CVE_CACHE_DIR)) mkdirSync(CVE_CACHE_DIR, { recursive: true });
  writeFileSync(CVE_CACHE_FILE, JSON.stringify(cves), "utf-8");

  const meta: CVEMeta = {
    last_updated: new Date().toISOString(),
    total: cves.length,
  };
  writeFileSync(CVE_META_FILE, JSON.stringify(meta, null, 2), "utf-8");

  cachedCVEs = cves;
  return cves;
}

function matchesQuery(cve: CVE, q: SearchQuery): boolean {
  if (q.search) {
    const s = q.search.toLowerCase();
    if (!cve.cve.toLowerCase().includes(s) && !cve.description.toLowerCase().includes(s)) return false;
  }
  if (q.vendor) {
    const v = q.vendor.toLowerCase();
    if (!cve.vendors?.some((x) => x.toLowerCase().includes(v))) return false;
  }
  if (q.severity && severity(cve) !== q.severity) return false;
  if (q.cisa !== undefined && cve.cisa !== q.cisa) return false;
  if (q.metasploit !== undefined && cve.metasploit !== q.metasploit) return false;

  const cvss = bestCVSS(cve);
  if (q.min_cvss !== undefined && cvss < q.min_cvss) return false;
  if (q.max_cvss !== undefined && cvss > q.max_cvss) return false;

  return true;
}

function sortCVEs(cves: CVE[], sortBy: string, desc: boolean) {
  cves.sort((a, b) => {
    let cmp = 0;
    switch (sortBy) {
      case "cvss": cmp = bestCVSS(a) - bestCVSS(b); break;
      case "epss": cmp = (a.epss ?? 0) - (b.epss ?? 0); break;
      case "vscore": cmp = (a.v_score ?? 0) - (b.v_score ?? 0); break;
      default:
        const da = a.published_datetime || "";
        const db = b.published_datetime || "";
        cmp = da.localeCompare(db);
        break;
    }
    return desc ? -cmp : cmp;
  });
}

/**
 * GET /api/talon/cve
 * Query params: search, vendor, severity, cisa, metasploit, min_cvss, max_cvss,
 *               sort_by (published|cvss|epss|vscore), sort_dir (asc|desc),
 *               page (default 1), size (default 50)
 */
export async function GET(req: NextRequest) {
  try {
    const { searchParams } = new URL(req.url);

    // Check if we need to download/refresh
    if (searchParams.get("refresh") === "true") {
      await downloadData();
      return NextResponse.json({ ok: true, message: "CVE data refreshed" });
    }

    let cves = loadCache();
    if (cves.length === 0) {
      cves = await downloadData();
    }

    // Parse query
    const q: SearchQuery = {
      search: searchParams.get("search") || undefined,
      vendor: searchParams.get("vendor") || undefined,
      severity: searchParams.get("severity") || undefined,
      cisa: searchParams.has("cisa") ? searchParams.get("cisa") === "true" : undefined,
      metasploit: searchParams.has("metasploit") ? searchParams.get("metasploit") === "true" : undefined,
      min_cvss: searchParams.has("min_cvss") ? parseFloat(searchParams.get("min_cvss")!) : undefined,
      max_cvss: searchParams.has("max_cvss") ? parseFloat(searchParams.get("max_cvss")!) : undefined,
      sort_by: searchParams.get("sort_by") || "published",
      sort_dir: searchParams.get("sort_dir") || "desc",
      page: parseInt(searchParams.get("page") || "1"),
      size: parseInt(searchParams.get("size") || "50"),
    };

    // Stats request
    if (searchParams.get("stats") === "true") {
      const stats = {
        total: cves.length,
        critical: cves.filter((c) => severity(c) === "critical").length,
        high: cves.filter((c) => severity(c) === "high").length,
        medium: cves.filter((c) => severity(c) === "medium").length,
        low: cves.filter((c) => severity(c) === "low").length,
        unknown: cves.filter((c) => severity(c) === "unknown").length,
        cisa: cves.filter((c) => c.cisa).length,
        metasploit: cves.filter((c) => c.metasploit).length,
        last_updated: getLastUpdated(cves),
      };
      return NextResponse.json(stats);
    }

    // Vendor list
    if (searchParams.get("vendors") === "true") {
      const vendorSet = new Set<string>();
      for (const c of cves) {
        for (const v of c.vendors || []) vendorSet.add(v);
      }
      return NextResponse.json({ vendors: Array.from(vendorSet).sort() });
    }

    // Filter
    let filtered = cves.filter((c) => matchesQuery(c, q));

    // Sort
    sortCVEs(filtered, q.sort_by!, q.sort_dir !== "asc");

    // Paginate
    const total = filtered.length;
    const page = Math.max(1, q.page!);
    const size = Math.min(1000, Math.max(1, q.size!));
    const start = (page - 1) * size;
    const end = start + size;
    const pageCVEs = filtered.slice(start, end);

    return NextResponse.json({
      total,
      page,
      size,
      cves: pageCVEs,
    });
  } catch (err) {
    const msg = err instanceof Error ? err.message : "CVE lookup failed";
    return NextResponse.json({ error: msg }, { status: 500 });
  }
}

function getLastUpdated(cves: CVE[]): string {
  let latest = "";
  for (const c of cves) {
    if (c.published_datetime && c.published_datetime > latest) latest = c.published_datetime;
  }
  return latest;
}
