"use client";

import { useState, useEffect, useCallback } from "react";
import { Search, RefreshCw, FolderOpen, Database } from "lucide-react";

const KG_PATH_KEY = "talon-kg-path";

// ─── Cache Tab ─────────────────────────────────────────────────────────

function CacheTab() {
  const [cacheStatus, setCacheStatus] = useState<string>("Checking...");
  const [key, setKey] = useState("");
  const [value, setValue] = useState("");
  const [lookupKey, setLookupKey] = useState("");
  const [lookupResult, setLookupResult] = useState<string | null>(null);
  const [ttl, setTtl] = useState("24h");

  useEffect(() => {
    fetch("/api/talon/cache")
      .then((r) => r.json())
      .then((data) => setCacheStatus(data.server?.status || "error"))
      .catch(() => setCacheStatus("unavailable"));
  }, []);

  const handleSet = async () => {
    if (!key || !value) return;
    try {
      const res = await fetch("/api/talon/cache", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key, value, ttl }),
      });
      const data = await res.json();
      setLookupResult(JSON.stringify(data, null, 2));
      setValue("");
    } catch (err: any) {
      setLookupResult(`Error: ${err.message}`);
    }
  };

  const handleGet = async () => {
    if (!lookupKey) return;
    try {
      const res = await fetch(`/api/talon/cache?key=${encodeURIComponent(lookupKey)}`);
      if (res.status === 404) {
        setLookupResult("Key not found");
        return;
      }
      const data = await res.json();
      setLookupResult(JSON.stringify(data, null, 2));
    } catch (err: any) {
      setLookupResult(`Error: ${err.message}`);
    }
  };

  const handleDelete = async () => {
    if (!lookupKey) return;
    try {
      const res = await fetch(`/api/talon/cache?key=${encodeURIComponent(lookupKey)}`, {
        method: "DELETE",
      });
      const data = await res.json();
      setLookupResult(JSON.stringify(data, null, 2));
    } catch (err: any) {
      setLookupResult(`Error: ${err.message}`);
    }
  };

  return (
    <div className="flex flex-col gap-4">
      {/* Status */}
      <div className="rounded-xl bg-base-200 border border-base-content/10 p-4 shadow-sm">
        <div className="flex items-center gap-2">
          <Database size={18} className={cacheStatus === "ok" ? "text-success" : "text-error"} />
          <span className="text-sm font-medium">Cache Server</span>
          <span className={`badge badge-xs ${cacheStatus === "ok" ? "badge-success" : "badge-error"}`}>
            {cacheStatus}
          </span>
        </div>
      </div>

      {/* Set */}
      <div className="rounded-xl bg-base-200 border border-base-content/10 p-4 shadow-sm">
        <h3 className="text-sm font-bold mb-3">Set Cache Entry</h3>
        <div className="flex flex-col gap-2">
          <input
            type="text"
            className="input input-bordered input-sm"
            placeholder="Key"
            value={key}
            onChange={(e) => setKey(e.target.value)}
          />
          <textarea
            className="textarea textarea-bordered textarea-sm font-mono text-xs"
            placeholder="Value (JSON string)"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            rows={3}
          />
          <div className="flex gap-2 items-center">
            <input
              type="text"
              className="input input-bordered input-sm w-32"
              placeholder="TTL (e.g. 24h)"
              value={ttl}
              onChange={(e) => setTtl(e.target.value)}
            />
            <button className="btn btn-primary btn-sm" onClick={handleSet}>Set</button>
          </div>
        </div>
      </div>

      {/* Get / Delete */}
      <div className="rounded-xl bg-base-200 border border-base-content/10 p-4 shadow-sm">
        <h3 className="text-sm font-bold mb-3">Lookup</h3>
        <div className="flex gap-2 items-center mb-3">
          <input
            type="text"
            className="input input-bordered input-sm flex-1"
            placeholder="Key to lookup"
            value={lookupKey}
            onChange={(e) => setLookupKey(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleGet()}
          />
          <button className="btn btn-primary btn-sm" onClick={handleGet}>Get</button>
          <button className="btn btn-outline btn-sm" onClick={handleDelete}>Delete</button>
        </div>
        {lookupResult && (
          <pre className="text-xs font-mono whitespace-pre-wrap text-base-content/80 bg-base-300 p-3 rounded-lg">
            {lookupResult}
          </pre>
        )}
      </div>
    </div>
  );
}

// ─── Graph Tab ─────────────────────────────────────────────────────────

function GraphTab({
  kgHtml,
  kgStatus,
  kgLoading,
  kgQuery,
  setKgQuery,
  kgResults,
  projectPath,
  setProjectPath,
  buildGraph,
  runKgQuery,
}: {
  kgHtml: string | null;
  kgStatus: string;
  kgLoading: boolean;
  kgQuery: string;
  setKgQuery: (q: string) => void;
  kgResults: string | null;
  projectPath: string;
  setProjectPath: (p: string) => void;
  buildGraph: () => void;
  runKgQuery: () => void;
}) {
  return (
    <div className="flex flex-col gap-6">
      {/* Header with build controls */}
      <div className="rounded-xl bg-base-200 border border-base-content/10 p-5 shadow-sm">
        <div className="flex items-center justify-between flex-wrap gap-3">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-lg bg-primary/15 flex items-center justify-center">
              <span className="material-symbols-outlined text-primary" style={{ fontSize: "18px" }}>hub</span>
            </div>
            <div>
              <div className="font-bold text-sm">Code Knowledge Graph</div>
              <div className="text-xs text-base-content/60">{kgStatus}</div>
            </div>
          </div>
          <button onClick={buildGraph} disabled={kgLoading} className="btn btn-primary btn-sm gap-2">
            <RefreshCw size={14} className={kgLoading ? "animate-spin" : ""} />
            {kgLoading ? "Building..." : "Build Graph"}
          </button>
        </div>

        {/* Project path input */}
        <div className="flex gap-2 mt-4">
          <label className="input input-bordered flex items-center gap-2 flex-1 bg-base-300 border-base-content/20">
            <FolderOpen size={14} className="text-base-content/60" />
            <input type="text" className="grow font-mono text-xs" placeholder="Path to codebase (e.g. /Users/me/my-project or . for server CWD)"
              value={projectPath} onChange={e => setProjectPath(e.target.value)}
              onKeyDown={e => e.key === "Enter" && buildGraph()} />
          </label>
        </div>

        {/* Query bar */}
        <div className="flex gap-2 mt-3">
          <label className="input input-bordered flex items-center gap-2 flex-1 bg-base-300 border-base-content/20">
            <Search size={14} className="text-base-content/60" />
            <input type="text" className="grow" placeholder="Search codebase (function, type, file...)" value={kgQuery}
              onChange={e => setKgQuery(e.target.value)}
              onKeyDown={e => e.key === "Enter" && runKgQuery()} />
          </label>
          <button onClick={runKgQuery} className="btn btn-primary btn-sm gap-1"><Search size={14} /> Query</button>
        </div>
      </div>

      {/* Query results */}
      {kgResults && (
        <div className="p-4 rounded-xl bg-base-200 border border-base-content/10 shadow-sm">
          <pre className="text-xs font-mono whitespace-pre-wrap text-base-content/80 leading-relaxed">{kgResults}</pre>
        </div>
      )}

      {/* Visualization */}
      <div className="rounded-xl bg-white border border-base-content/10 overflow-hidden shadow-sm min-h-[400px] relative">
        {kgHtml ? (
          <iframe srcDoc={kgHtml} className="w-full h-[600px]" title="Knowledge Graph" sandbox="allow-scripts allow-same-origin" />
        ) : (
          <div className="absolute inset-0 flex items-center justify-center text-base-content/30 text-sm">
            <div className="text-center">
              <div className="text-5xl mb-3 opacity-50">&#9670;</div>
              <p className="text-base font-medium">No knowledge graph yet</p>
              <p className="mt-1">Click "Build Graph" to analyse your codebase</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Main Page ─────────────────────────────────────────────────────────

const TABS = [
  { id: "graph", label: "Graph", icon: "hub" },
  { id: "cache", label: "Cache", icon: "database" },
] as const;

type TabId = (typeof TABS)[number]["id"];

export default function MemoryPage() {
  const [activeTab, setActiveTab] = useState<TabId>("graph");
  const [kgHtml, setKgHtml] = useState<string | null>(null);
  const [kgStatus, setKgStatus] = useState<string>("No graph loaded");
  const [kgLoading, setKgLoading] = useState(false);
  const [kgQuery, setKgQuery] = useState("");
  const [kgResults, setKgResults] = useState<string | null>(null);
  const [projectPath, setProjectPath] = useState<string>("");

  // Load saved project path from localStorage
  useEffect(() => {
    const saved = localStorage.getItem(KG_PATH_KEY);
    if (saved) setProjectPath(saved);
  }, []);

  // Persist project path to localStorage whenever it changes
  useEffect(() => {
    if (projectPath) localStorage.setItem(KG_PATH_KEY, projectPath);
  }, [projectPath]);

  // Load knowledge graph status on mount
  useEffect(() => {
    fetch("/api/talon/knowledge")
      .then(r => r.json())
      .then(data => {
        if (data.exists) {
          setKgHtml(data.html);
          setKgStatus(`Loaded (built ${new Date(data.lastBuilt).toLocaleString()})`);
        }
      })
      .catch(() => {});
  }, []);

  const buildGraph = useCallback(async () => {
    setKgLoading(true);
    setKgStatus("Building graph...");
    try {
      const res = await fetch("/api/talon/knowledge", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "build", path: projectPath || "." }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error);
      const resolvedPath = projectPath || ".";
      setKgStatus(`Graph built — analysing "${resolvedPath}"`);

      const vizRes = await fetch("/api/talon/knowledge", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "visualize" }),
      });
      const vizData = await vizRes.json();
      if (vizData.html) setKgHtml(vizData.html);
    } catch (err: any) {
      setKgStatus(`Error: ${err.message}`);
    } finally {
      setKgLoading(false);
    }
  }, [projectPath]);

  const runKgQuery = useCallback(async () => {
    if (!kgQuery) return;
    setKgResults("Querying...");
    try {
      const res = await fetch("/api/talon/knowledge", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "query", queryType: "search", param1: kgQuery, param2: "" }),
      });
      const data = await res.json();
      setKgResults(data.result || "No results");
    } catch (err: any) {
      setKgResults(`Error: ${err.message}`);
    }
  }, [kgQuery]);

  return (
    <div className="flex flex-col gap-4">
      {/* Tabs */}
      <div className="flex gap-1 p-1 rounded-xl bg-base-200 border border-base-content/10 w-fit">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              activeTab === tab.id
                ? "bg-primary text-primary-content shadow-sm"
                : "text-base-content/60 hover:text-base-content hover:bg-base-300/50"
            }`}
          >
            <span className="material-symbols-outlined" style={{ fontSize: "16px" }}>{tab.icon}</span>
            {tab.label}
          </button>
        ))}
      </div>

      {/* Content */}
      {activeTab === "graph" ? (
        <GraphTab
          kgHtml={kgHtml}
          kgStatus={kgStatus}
          kgLoading={kgLoading}
          kgQuery={kgQuery}
          setKgQuery={setKgQuery}
          kgResults={kgResults}
          projectPath={projectPath}
          setProjectPath={setProjectPath}
          buildGraph={buildGraph}
          runKgQuery={runKgQuery}
        />
      ) : (
        <CacheTab />
      )}
    </div>
  );
}
