"use client";

import { useEffect, useState, useCallback } from "react";
import {
  Search,
  Shield,
  Filter,
  ChevronDown,
  ChevronUp,
  Download,
  RefreshCw,
  AlertTriangle,
  Bug,
  ExternalLink,
} from "lucide-react";

interface CVE {
  cve: string;
  description: string;
  vendors: string[];
  cvssv3: number | null;
  cvssv2: number | null;
  epss: number | null;
  v_score: number | null;
  published_datetime: string;
  cisa: boolean;
  metasploit: boolean;
  cti_count: number;
  social_media_audience: number;
}

interface Stats {
  total: number;
  critical: number;
  high: number;
  medium: number;
  low: number;
  unknown: number;
  cisa: number;
  metasploit: number;
  last_updated: string;
}

function severityClass(sev: string) {
  switch (sev) {
    case "critical": return "badge-error";
    case "high": return "badge-warning";
    case "medium": return "badge-info";
    case "low": return "badge-ghost";
    default: return "badge-ghost";
  }
}

function severityLabel(cve: CVE): string {
  const s = cve.cvssv3 ?? cve.cvssv2;
  if (s == null) return "unknown";
  if (s >= 9.0) return "critical";
  if (s >= 7.0) return "high";
  if (s >= 4.0) return "medium";
  if (s > 0) return "low";
  return "unknown";
}

function formatDate(d: string) {
  if (!d) return "-";
  return new Date(d).toLocaleDateString("en-US", {
    year: "numeric", month: "short", day: "numeric",
  });
}

function cvssColor(s: number | null) {
  if (s == null) return "text-base-content/40";
  if (s >= 9.0) return "text-error";
  if (s >= 7.0) return "text-warning";
  if (s >= 4.0) return "text-info";
  return "text-base-content/60";
}

export default function SecurityPage() {
  const [cves, setCVEs] = useState<CVE[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(50);

  // Filters
  const [search, setSearch] = useState("");
  const [vendor, setVendor] = useState("");
  const [severityFilter, setSeverityFilter] = useState("");
  const [cisaOnly, setCisaOnly] = useState(false);
  const [metaOnly, setMetaOnly] = useState(false);
  const [sortBy, setSortBy] = useState("published");
  const [sortDir, setSortDir] = useState("desc");

  const [expanded, setExpanded] = useState<string | null>(null);
  const [initialLoad, setInitialLoad] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchCVEs = useCallback(async (p: number) => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (search) params.set("search", search);
      if (vendor) params.set("vendor", vendor);
      if (severityFilter) params.set("severity", severityFilter);
      if (cisaOnly) params.set("cisa", "true");
      if (metaOnly) params.set("metasploit", "true");
      params.set("sort_by", sortBy);
      params.set("sort_dir", sortDir);
      params.set("page", String(p));
      params.set("size", String(pageSize));

      const res = await fetch(`/api/talon/cve?${params}`);
      if (!res.ok) {
        const errData = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
        throw new Error(errData.error || `HTTP ${res.status}`);
      }
      const data = await res.json();
      setCVEs(data.cves || []);
      setTotal(data.total || 0);
    } catch (err: any) {
      setError(err.message || "Failed to load CVEs");
    } finally {
      setLoading(false);
    }
  }, [search, vendor, severityFilter, cisaOnly, metaOnly, sortBy, sortDir, pageSize]);

  const fetchStats = useCallback(async () => {
    try {
      const res = await fetch("/api/talon/cve?stats=true");
      if (res.ok) {
        setStats(await res.json());
      }
    } catch {}
  }, []);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  useEffect(() => {
    if (initialLoad) {
      setInitialLoad(false);
      return;
    }
    setPage(1);
    fetchCVEs(1);
  }, [search, vendor, severityFilter, cisaOnly, metaOnly, sortBy, sortDir, initialLoad, fetchCVEs]);

  useEffect(() => {
    fetchCVEs(page);
  }, [page, fetchCVEs]);

  const totalPages = Math.ceil(total / pageSize);

  const toggleSort = (field: string) => {
    if (sortBy === field) {
      setSortDir((d) => (d === "desc" ? "asc" : "desc"));
    } else {
      setSortBy(field);
      setSortDir("desc");
    }
  };

  const refreshData = async () => {
    setLoading(true);
    try {
      await fetch("/api/talon/cve?refresh=true");
      await fetchStats();
      await fetchCVEs(1);
    } catch {} finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Shield size={28} className="text-error" />
          <div>
            <h1 className="text-2xl font-bold">Security</h1>
            <p className="text-sm text-base-content/60">
              CVE vulnerability database — {stats?.total?.toLocaleString() ?? "..."} entries tracked
            </p>
          </div>
        </div>
        <button className="btn btn-outline btn-sm gap-2" onClick={refreshData} disabled={loading}>
          <RefreshCw size={14} className={loading ? "animate-spin" : ""} />
          Refresh
        </button>
      </div>

      {/* Stats cards */}
      {stats && (
        <div className="grid grid-cols-5 gap-3">
          <StatCard label="Critical" value={stats.critical} color="text-error" bg="bg-error/10" />
          <StatCard label="High" value={stats.high} color="text-warning" bg="bg-warning/10" />
          <StatCard label="Medium" value={stats.medium} color="text-info" bg="bg-info/10" />
          <StatCard label="CISA KEV" value={stats.cisa} color="text-error" bg="bg-error/10" icon={<AlertTriangle size={16} />} />
          <StatCard label="Metasploit" value={stats.metasploit} color="text-warning" bg="bg-warning/10" icon={<Bug size={16} />} />
        </div>
      )}

      {/* Error banner */}
      {error && (
        <div className="alert alert-error text-sm">
          <span>{error}</span>
          <button className="btn btn-ghost btn-xs" onClick={() => { setError(null); fetchCVEs(1); }}>Retry</button>
        </div>
      )}

      {/* Filters */}
      <div className="flex flex-wrap gap-3 items-end">
        <div className="form-control flex-1 min-w-[200px]">
          <label className="input input-bordered input-sm flex items-center gap-2">
            <Search size={14} className="text-base-content/40" />
            <input
              type="text"
              className="grow"
              placeholder="Search CVE ID or description..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </label>
        </div>

        <div className="form-control">
          <input
            type="text"
            className="input input-bordered input-sm"
            placeholder="Vendor..."
            value={vendor}
            onChange={(e) => setVendor(e.target.value)}
          />
        </div>

        <div className="form-control">
          <select
            className="select select-bordered select-sm"
            value={severityFilter}
            onChange={(e) => setSeverityFilter(e.target.value)}
          >
            <option value="">All Severities</option>
            <option value="critical">Critical</option>
            <option value="high">High</option>
            <option value="medium">Medium</option>
            <option value="low">Low</option>
          </select>
        </div>

        <label className="label cursor-pointer gap-2">
          <input
            type="checkbox"
            className="checkbox checkbox-xs"
            checked={cisaOnly}
            onChange={(e) => setCisaOnly(e.target.checked)}
          />
          <span className="label-text text-xs">CISA KEV</span>
        </label>

        <label className="label cursor-pointer gap-2">
          <input
            type="checkbox"
            className="checkbox checkbox-xs"
            checked={metaOnly}
            onChange={(e) => setMetaOnly(e.target.checked)}
          />
          <span className="label-text text-xs">Metasploit</span>
        </label>
      </div>

      {/* Result count */}
      <div className="text-sm text-base-content/50">
        {loading ? "Searching..." : `${total.toLocaleString()} results`}
      </div>

      {/* Table */}
      <div className="overflow-x-auto">
        <table className="table table-zebra table-pin-rows text-xs">
          <thead>
            <tr>
              <th className="w-[140px] cursor-pointer" onClick={() => toggleSort("published")}>
                CVE ID {sortBy === "published" && (sortDir === "desc" ? <ChevronDown size={12} className="inline" /> : <ChevronUp size={12} className="inline" />)}
              </th>
              <th className="cursor-pointer" onClick={() => toggleSort("cvss")}>
                CVSS {sortBy === "cvss" && (sortDir === "desc" ? <ChevronDown size={12} className="inline" /> : <ChevronUp size={12} className="inline" />)}
              </th>
              <th>Severity</th>
              <th className="cursor-pointer" onClick={() => toggleSort("epss")}>
                EPSS {sortBy === "epss" && (sortDir === "desc" ? <ChevronDown size={12} className="inline" /> : <ChevronUp size={12} className="inline" />)}
              </th>
              <th>Description</th>
              <th>Vendors</th>
              <th>Flags</th>
              <th>Published</th>
            </tr>
          </thead>
          <tbody>
            {cves.map((cve) => (
              <tr
                key={cve.cve}
                className="cursor-pointer"
                onClick={() => setExpanded(expanded === cve.cve ? null : cve.cve)}
              >
                <td className="font-mono">
                  <a
                    href={`https://nvd.nist.gov/vuln/detail/${cve.cve}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="link link-hover"
                    onClick={(e) => e.stopPropagation()}
                  >
                    {cve.cve}
                  </a>
                </td>
                <td className={`font-mono ${cvssColor(cve.cvssv3 ?? cve.cvssv2)}`}>
                  {(cve.cvssv3 ?? cve.cvssv2 ?? "-")}
                </td>
                <td>
                  <span className={`badge badge-sm ${severityClass(severityLabel(cve))}`}>
                    {severityLabel(cve)}
                  </span>
                </td>
                <td className="font-mono">
                  {cve.epss != null ? `${(cve.epss * 100).toFixed(1)}%` : "-"}
                </td>
                <td className="max-w-md truncate">{cve.description}</td>
                <td className="max-w-[150px] truncate">{cve.vendors?.slice(0, 2).join(", ") || "-"}</td>
                <td>
                  <div className="flex gap-1">
                    {cve.cisa && <span className="badge badge-xs badge-error" title="CISA KEV">CISA</span>}
                    {cve.metasploit && <span className="badge badge-xs badge-warning" title="Metasploit">MSF</span>}
                  </div>
                </td>
                <td className="text-base-content/60">{formatDate(cve.published_datetime)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex justify-center gap-2">
          <button
            className="btn btn-ghost btn-xs"
            disabled={page <= 1}
            onClick={() => setPage((p) => Math.max(1, p - 1))}
          >
            Previous
          </button>
          <span className="text-sm text-base-content/60 self-center">
            Page {page} of {totalPages}
          </span>
          <button
            className="btn btn-ghost btn-xs"
            disabled={page >= totalPages}
            onClick={() => setPage((p) => p + 1)}
          >
            Next
          </button>
        </div>
      )}

      {!loading && cves.length === 0 && (
        <div className="text-center py-12 text-base-content/40">
          <Shield size={48} className="mx-auto mb-3 opacity-30" />
          <p>No CVEs match your filters.</p>
        </div>
      )}
    </div>
  );
}

function StatCard({
  label, value, color, bg, icon,
}: {
  label: string; value: number; color: string; bg: string; icon?: React.ReactNode;
}) {
  return (
    <div className={`${bg} rounded-lg p-3`}>
      <div className="flex items-center gap-2">
        {icon}
        <div className={`text-2xl font-bold ${color}`}>{value.toLocaleString()}</div>
      </div>
      <div className="text-xs text-base-content/50 mt-1">{label}</div>
    </div>
  );
}
