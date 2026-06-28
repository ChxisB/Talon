"use client";

import { useState, useRef, useCallback } from "react";
import { PanelRightOpen, Download, RefreshCw } from "lucide-react";

// ─── Diagram Builder presets ──────────────────────────────────────────

const DIAGRAM_TYPES = [
  { id: "architecture", label: "Architecture", desc: "System components, cloud infra, services, security boundaries" },
  { id: "workflow", label: "Workflow", desc: "CI/CD, approval flows, tool calls, runbooks" },
  { id: "sequence", label: "Sequence", desc: "API call chains, request lifecycles, async traces" },
  { id: "dataflow", label: "Data Flow", desc: "Pipelines, ETL, data lineage, PII boundaries" },
  { id: "lifecycle", label: "Lifecycle", desc: "State machines, status transitions, retry logic" },
];

const EXAMPLE_JSON: Record<string, string> = {
  architecture: JSON.stringify({
    schema_version: 1,
    diagram_type: "architecture",
    meta: { title: "Web App", subtitle: "3-tier architecture" },
    components: [
      { id: "users", type: "external", label: "Users", sublabel: "Browser", pos: [40, 250] },
      { id: "cdn", type: "cloud", label: "CloudFront", sublabel: "CDN", pos: [200, 250] },
      { id: "api", type: "backend", label: "API Server", sublabel: "FastAPI", pos: [380, 250] },
      { id: "db", type: "database", label: "PostgreSQL", sublabel: "Primary", pos: [560, 250] },
    ],
    connections: [
      { from: "users", to: "cdn", label: "HTTPS", variant: "emphasis" },
      { from: "cdn", to: "api", label: "Origin" },
      { from: "api", to: "db", label: "SQL" },
    ],
    boundaries: [
      { kind: "region", label: "AWS", wraps: ["cdn", "api", "db"] },
    ],
  }, null, 2),
  workflow: JSON.stringify({
    schema_version: 1,
    diagram_type: "workflow",
    meta: { title: "CI/CD Pipeline", subtitle: "PR to production" },
    lanes: [
      { id: "dev", label: "Developer" },
      { id: "ci", label: "CI/CD" },
    ],
    nodes: [
      { id: "pr", lane: "dev", col: 0, type: "frontend", label: "Open PR" },
      { id: "build", lane: "ci", col: 1, type: "backend", label: "Build & Test" },
      { id: "deploy", lane: "ci", col: 2, type: "cloud", label: "Deploy" },
    ],
    edges: [
      { from: "pr", to: "build", label: "push", variant: "emphasis" },
      { from: "build", to: "deploy", label: "auto" },
    ],
  }, null, 2),
  sequence: JSON.stringify({
    schema_version: 1,
    diagram_type: "sequence",
    meta: { title: "Login Flow", subtitle: "OAuth2 authentication" },
    participants: [
      { id: "browser", type: "frontend", label: "Browser" },
      { id: "api", type: "backend", label: "API Server" },
      { id: "auth", type: "security", label: "Auth Provider" },
      { id: "db", type: "database", label: "Database" },
    ],
    messages: [
      { from: "browser", to: "api", y: 200, label: "POST /login" },
      { from: "api", to: "auth", y: 260, label: "verify", variant: "security" },
      { from: "auth", to: "api", y: 320, label: "token", variant: "return" },
      { from: "api", to: "db", y: 380, label: "store session" },
      { from: "api", to: "browser", y: 440, label: "200 OK", variant: "return" },
    ],
  }, null, 2),
  dataflow: JSON.stringify({
    schema_version: 1,
    diagram_type: "dataflow",
    meta: { title: "Analytics Pipeline", subtitle: "Events to warehouse" },
    stages: [
      { label: "Sources" },
      { label: "Ingest" },
      { label: "Process" },
      { label: "Store" },
    ],
    nodes: [
      { id: "web", type: "frontend", label: "Web App", stage: 0, row: 0 },
      { id: "mobile", type: "frontend", label: "Mobile App", stage: 0, row: 1 },
      { id: "kafka", type: "messagebus", label: "Event Stream", stage: 1, row: 0 },
      { id: "etl", type: "backend", label: "ETL Pipeline", stage: 2, row: 0 },
      { id: "warehouse", type: "database", label: "Data Warehouse", stage: 3, row: 0 },
    ],
    flows: [
      { from: "web", to: "kafka", label: "clickstream" },
      { from: "mobile", to: "kafka", label: "app events" },
      { from: "kafka", to: "etl", label: "stream" },
      { from: "etl", to: "warehouse", label: "load" },
    ],
  }, null, 2),
  lifecycle: JSON.stringify({
    schema_version: 1,
    diagram_type: "lifecycle",
    meta: { title: "Agent Run Lifecycle", subtitle: "State machine" },
    lanes: [
      { id: "phase", label: "Execution Phase" },
      { id: "waiting", label: "Waiting States" },
      { id: "terminal", label: "Terminal" },
    ],
    states: [
      { id: "queued", type: "start", label: "Queued", lane: "phase", col: 0, step: "01" },
      { id: "running", type: "active", label: "Running", lane: "phase", col: 1, step: "02" },
      { id: "approval", type: "waiting", label: "Needs Approval", lane: "waiting", col: 0 },
      { id: "done", type: "success", label: "Completed", lane: "terminal", col: 0 },
      { id: "failed", type: "failure", label: "Failed", lane: "terminal", col: 1 },
    ],
    transitions: [
      { from: "queued", to: "running", variant: "emphasis" },
      { from: "running", to: "approval", label: "requires approval", variant: "security" },
      { from: "approval", to: "running", label: "approved" },
      { from: "running", to: "done", label: "success" },
      { from: "running", to: "failed", label: "error" },
    ],
  }, null, 2),
};

// ─── Page ────────────────────────────────────────────────────────────

export default function DiagramsPage() {
  const [diagramType, setDiagramType] = useState("architecture");
  const [jsonInput, setJsonInput] = useState(EXAMPLE_JSON["architecture"]);
  const [htmlPreview, setHtmlPreview] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [diagramPath, setDiagramPath] = useState<string | null>(null);
  const iframeRef = useRef<HTMLIFrameElement>(null);

  const loadExample = useCallback(() => {
    setJsonInput(EXAMPLE_JSON[diagramType] || EXAMPLE_JSON["architecture"]);
    setHtmlPreview(null);
    setError(null);
  }, [diagramType]);

  const generate = useCallback(async () => {
    setLoading(true);
    setError(null);
    setHtmlPreview(null);

    try {
      const res = await fetch("/api/talon/diagrams/generate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ type: diagramType, json: jsonInput }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Failed to generate diagram");

      setHtmlPreview(data.html);
      setDiagramPath(data.path);
    } catch (err: any) {
      setError(err.message || String(err));
    } finally {
      setLoading(false);
    }
  }, [diagramType, jsonInput]);

  const download = useCallback(() => {
    if (!htmlPreview) return;
    const blob = new Blob([htmlPreview], { type: "text/html" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${diagramType}-diagram.html`;
    a.click();
    URL.revokeObjectURL(url);
  }, [htmlPreview, diagramType]);

  return (
    <div className="flex flex-col gap-6 p-6 max-w-full">
      {/* Header */}
      <div className="flex items-center gap-3">
        <div className="w-1 h-6 rounded-full bg-primary" />
        <h1 className="font-bold text-xl tracking-tight">Diagram Builder</h1>
        <span className="text-xs text-base-content/70">Generate architecture diagrams with instant preview</span>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
        {/* Left: Editor */}
        <div className="flex flex-col gap-4">
          {/* Type selector */}
          <div className="flex flex-wrap gap-2">
            {DIAGRAM_TYPES.map((dt) => (
              <button
                key={dt.id}
                onClick={() => { setDiagramType(dt.id); setHtmlPreview(null); setError(null); }}
                className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all border ${
                  diagramType === dt.id
                    ? "bg-primary text-primary-content border-primary"
                    : "bg-base-200 text-base-content/70 border-base-content/10 hover:border-base-content/30"
                }`}
                title={dt.desc}
              >
                {dt.label}
              </button>
            ))}
          </div>

          {/* JSON editor */}
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-base-content/70">Diagram JSON Spec</span>
            <button
              onClick={loadExample}
              className="text-xs text-primary hover:underline"
            >
              Load example
            </button>
          </div>
          <textarea
            value={jsonInput}
            onChange={(e) => setJsonInput(e.target.value)}
            className="w-full h-[400px] font-mono text-xs p-3 rounded-xl bg-base-300 border border-base-content/10 focus:outline-none focus:border-primary resize-y"
            spellCheck={false}
          />

          {/* Actions */}
          <div className="flex gap-2">
            <button
              onClick={generate}
              disabled={loading}
              className="flex items-center gap-2 px-4 py-2 rounded-xl bg-primary text-primary-content font-medium text-sm hover:opacity-90 disabled:opacity-50 transition-all"
            >
              {loading ? <RefreshCw className="w-4 h-4 animate-spin" /> : <PanelRightOpen className="w-4 h-4" />}
              {loading ? "Generating..." : "Generate"}
            </button>
            {htmlPreview && (
              <button
                onClick={download}
                className="flex items-center gap-2 px-4 py-2 rounded-xl bg-base-200 border border-base-content/10 text-sm hover:bg-base-300 transition-all"
              >
                <Download className="w-4 h-4" />
                Download HTML
              </button>
            )}
            {diagramPath && (
              <span className="text-xs text-base-content/50 self-center ml-2 truncate max-w-[200px]" title={diagramPath}>
                {diagramPath}
              </span>
            )}
          </div>

          {/* Error */}
          {error && (
            <div className="p-3 rounded-xl bg-error/10 border border-error/30 text-error text-sm">
              {error}
            </div>
          )}
        </div>

        {/* Right: Preview */}
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium text-base-content/70">Preview</span>
            {htmlPreview && (
              <span className="text-xs text-base-content/50">— diagram renders with dark/light toggle & export menu</span>
            )}
          </div>
          <div className="flex-1 min-h-[500px] rounded-xl bg-white border border-base-content/10 overflow-hidden relative">
            {htmlPreview ? (
              <iframe
                ref={iframeRef}
                srcDoc={htmlPreview}
                className="w-full h-full"
                title="Diagram Preview"
                sandbox="allow-scripts allow-same-origin"
              />
            ) : (
              <div className="absolute inset-0 flex items-center justify-center text-base-content/30 text-sm">
                <div className="text-center">
                  <div className="text-4xl mb-2">&#9736;</div>
                  <p>Enter a diagram JSON spec and click Generate</p>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
