"use client";

import { useState, useEffect, useCallback } from "react";
import { RefreshCw, Search, Network } from "lucide-react";

export default function KnowledgePage() {
  const [html, setHtml] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [graphStatus, setGraphStatus] = useState<string>("No graph loaded");
  const [queryResults, setQueryResults] = useState<string | null>(null);
  const [queryMode, setQueryMode] = useState<"neighbors" | "search" | "path">("neighbors");
  const [queryParam1, setQueryParam1] = useState("");
  const [queryParam2, setQueryParam2] = useState("");

  // Load graph status on mount
  useEffect(() => {
    fetch("/api/talon/knowledge")
      .then(r => r.json())
      .then(data => {
        if (data.exists) {
          setHtml(data.html);
          setGraphStatus(`Graph loaded (built ${new Date(data.lastBuilt).toLocaleString()})`);
        }
      })
      .catch(() => {});
  }, []);

  const buildGraph = useCallback(async () => {
    setLoading(true);
    setGraphStatus("Building graph...");
    try {
      const res = await fetch("/api/talon/knowledge", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "build", path: "." }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error);
      setGraphStatus(data.result || "Graph built");

      // Load the visualization
      const vizRes = await fetch("/api/talon/knowledge", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "visualize" }),
      });
      const vizData = await vizRes.json();
      if (vizData.html) setHtml(vizData.html);
    } catch (err: any) {
      setGraphStatus(`Error: ${err.message}`);
    } finally {
      setLoading(false);
    }
  }, []);

  const runQuery = useCallback(async () => {
    if (!queryParam1) return;
    setQueryResults("Querying...");
    try {
      const res = await fetch("/api/talon/knowledge", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action: "query",
          queryType: queryMode,
          param1: queryParam1,
          param2: queryMode === "path" ? queryParam2 : "",
        }),
      });
      const data = await res.json();
      setQueryResults(data.result || "No results");
    } catch (err: any) {
      setQueryResults(`Error: ${err.message}`);
    }
  }, [queryMode, queryParam1, queryParam2]);

  const refreshViz = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/talon/knowledge", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: "visualize" }),
      });
      const data = await res.json();
      if (data.html) setHtml(data.html);
      setGraphStatus(data.result || "Visualization refreshed");
    } catch (err: any) {
      setGraphStatus(`Error: ${err.message}`);
    } finally {
      setLoading(false);
    }
  }, []);

  return (
    <div className="flex flex-col gap-6 p-6 max-w-full">
      {/* Header */}
      <div className="flex items-center gap-3">
        <div className="w-1 h-6 rounded-full bg-primary" />
        <h1 className="font-bold text-xl tracking-tight">Knowledge Graph</h1>
        <span className="text-xs text-base-content/70">Codebase graph &mdash; find connections without reading files</span>
      </div>

      {/* Status + actions */}
      <div className="flex items-center gap-3 flex-wrap">
        <span className="text-xs text-base-content/70">{graphStatus}</span>
        <button
          onClick={buildGraph}
          disabled={loading}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-primary text-primary-content text-xs font-medium hover:opacity-90 disabled:opacity-50"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin" : ""}`} />
          Build Graph
        </button>
        <button
          onClick={refreshViz}
          disabled={loading || !html}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-base-200 border border-base-content/10 text-xs hover:bg-base-300 disabled:opacity-50"
        >
          <Network className="w-3.5 h-3.5" />
          Refresh Viz
        </button>
      </div>

      {/* Query panel */}
      <div className="flex flex-wrap items-end gap-3 p-4 rounded-xl bg-base-200 border border-base-content/10">
        <div>
          <label className="text-xs text-base-content/70 block mb-1">Query type</label>
          <select
            value={queryMode}
            onChange={e => setQueryMode(e.target.value as any)}
            className="px-2 py-1.5 rounded-lg bg-base-300 border border-base-content/10 text-xs"
          >
            <option value="neighbors">Neighbors</option>
            <option value="search">Search</option>
            <option value="path">Path</option>
          </select>
        </div>
        <div>
          <label className="text-xs text-base-content/70 block mb-1">
            {queryMode === "search" ? "Search term" : "Node ID"}
          </label>
          <input
            value={queryParam1}
            onChange={e => setQueryParam1(e.target.value)}
            placeholder={queryMode === "search" ? "e.g. saver" : "e.g. func:GenerateDiagram"}
            className="px-2 py-1.5 rounded-lg bg-base-300 border border-base-content/10 text-xs w-48"
            onKeyDown={e => e.key === "Enter" && runQuery()}
          />
        </div>
        {queryMode === "path" && (
          <div>
            <label className="text-xs text-base-content/70 block mb-1">To node ID</label>
            <input
              value={queryParam2}
              onChange={e => setQueryParam2(e.target.value)}
              placeholder="e.g. func:InitSaver"
              className="px-2 py-1.5 rounded-lg bg-base-300 border border-base-content/10 text-xs w-48"
              onKeyDown={e => e.key === "Enter" && runQuery()}
            />
          </div>
        )}
        <button
          onClick={runQuery}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-primary text-primary-content text-xs font-medium hover:opacity-90"
        >
          <Search className="w-3.5 h-3.5" />
          Query
        </button>
      </div>

      {/* Query results */}
      {queryResults && (
        <div className="p-4 rounded-xl bg-base-200 border border-base-content/10">
          <pre className="text-xs font-mono whitespace-pre-wrap text-base-content/80">{queryResults}</pre>
        </div>
      )}

      {/* Graph visualization */}
      <div className="rounded-xl bg-white border border-base-content/10 overflow-hidden min-h-[400px] relative">
        {html ? (
          <iframe
            srcDoc={html}
            className="w-full h-[600px]"
            title="Knowledge Graph"
            sandbox="allow-scripts allow-same-origin"
          />
        ) : (
          <div className="absolute inset-0 flex items-center justify-center text-base-content/30 text-sm">
            <div className="text-center">
              <div className="text-4xl mb-2">&#9673;</div>
              <p>Click "Build Graph" to analyse your codebase</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
