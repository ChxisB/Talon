"use client";

import { useEffect, useRef, useState, useCallback, useMemo } from "react";

// ─── Types ─────────────────────────────────────────────────────────

interface ModelEntry { id: string; provider: string; }
interface EchoSettings { config: Record<string, string>; }

// ─── Pricing table (per 1M tokens, USD) ────────────────────────────

const PRICING: Record<string, { input: number; output: number; context: number }> = {
  "gpt-4o":           { input: 2.50,  output: 10.00, context: 128000 },
  "gpt-4o-mini":      { input: 0.15,  output: 0.60,  context: 128000 },
  "o3":               { input: 10.00, output: 40.00, context: 200000 },
  "o4-mini":          { input: 1.10,  output: 4.40,  context: 200000 },
  "gpt-5.5":          { input: 15.00, output: 75.00, context: 200000 },
  "gpt-5.5-pro":      { input: 25.00, output: 100.00, context: 200000 },
  "gpt-5.4":          { input: 10.00, output: 40.00, context: 200000 },
  "gpt-5.4-pro":      { input: 15.00, output: 75.00, context: 200000 },
  "gpt-5.4-mini":     { input: 2.50,  output: 10.00, context: 128000 },
  "gpt-5.4-nano":     { input: 0.15,  output: 0.60,  context: 128000 },
  "gpt-5.3-codex":    { input: 10.00, output: 40.00, context: 200000 },
  "gpt-5.2":          { input: 2.50,  output: 10.00, context: 128000 },
  "gpt-5.1":          { input: 2.50,  output: 10.00, context: 128000 },
  "gpt-5":            { input: 2.50,  output: 10.00, context: 128000 },
  "claude-sonnet-4":  { input: 3.00,  output: 15.00, context: 200000 },
  "claude-sonnet-4.5":{ input: 3.00,  output: 15.00, context: 200000 },
  "claude-sonnet-4.6":{ input: 3.00,  output: 15.00, context: 200000 },
  "claude-haiku-4":   { input: 0.25,  output: 1.25,  context: 200000 },
  "claude-haiku-4.5": { input: 0.25,  output: 1.25,  context: 200000 },
  "claude-opus-4":    { input: 15.00, output: 75.00, context: 200000 },
  "claude-opus-4.1":  { input: 15.00, output: 75.00, context: 200000 },
  "claude-opus-4.5":  { input: 15.00, output: 75.00, context: 200000 },
  "claude-opus-4.6":  { input: 15.00, output: 75.00, context: 200000 },
  "claude-opus-4.7":  { input: 15.00, output: 75.00, context: 200000 },
  "claude-opus-4.8":  { input: 15.00, output: 75.00, context: 200000 },
  "claude-fable-5":   { input: 3.00,  output: 15.00, context: 200000 },
  "claude-3-5-haiku": { input: 0.80,  output: 4.00,  context: 200000 },
  "gemini-2.0-flash":  { input: 0.075, output: 0.30, context: 1000000 },
  "gemini-2.0-pro":    { input: 1.25,  output: 5.00, context: 2000000 },
  "gemini-2.5-flash":  { input: 0.15,  output: 0.60, context: 1000000 },
  "gemini-2.5-pro":    { input: 1.25,  output: 10.00, context: 2000000 },
  "gemini-3-flash":    { input: 0.15,  output: 0.60, context: 1000000 },
  "gemini-3.1-pro":    { input: 1.25,  output: 10.00, context: 2000000 },
  "gemini-3.5-flash":  { input: 0.15,  output: 0.60, context: 1000000 },
  "deepseek-chat":     { input: 0.15,  output: 0.60, context: 128000 },
  "deepseek-v4-pro":   { input: 3.45,  output: 8.55, context: 128000 },
  "deepseek-v4-flash": { input: 0.05,  output: 0.15, context: 128000 },
  "glm-5.1":           { input: 0.88,  output: 2.15, context: 128000 },
  "glm-5":             { input: 1.15,  output: 2.88, context: 128000 },
  "kimi-k2.5":         { input: 1.85,  output: 4.63, context: 128000 },
  "kimi-k2.6":         { input: 1.15,  output: 2.88, context: 128000 },
  "mimo-v2.5":         { input: 30.10, output: 75.20, context: 128000 },
  "mimo-v2.5-pro":     { input: 3.25,  output: 8.15, context: 128000 },
  "minimax-m3":        { input: 3.20,  output: 8.00, context: 128000 },
  "minimax-m2.7":      { input: 3.40,  output: 8.50, context: 128000 },
  "minimax-m2.5":      { input: 6.30,  output: 15.90, context: 128000 },
  "qwen3.7-max":       { input: 0.95,  output: 2.39, context: 128000 },
  "qwen3.7-plus":      { input: 4.30,  output: 10.80, context: 128000 },
  "qwen3.6-plus":      { input: 3.30,  output: 8.20, context: 128000 },
  "mistral-small":     { input: 0.20,  output: 0.60, context: 32000 },
  "mistral-large":     { input: 2.00,  output: 6.00, context: 128000 },
  "mixtral-8x7b":      { input: 0.00,  output: 0.00, context: 32000 },
  "llama-3.3-70b":     { input: 0.00,  output: 0.00, context: 128000 },
};

const PROVIDER_DISPLAY: Record<string, { label: string; color: string; docs?: string }> = {
  openai:       { label: "OpenAI",        color: "#10a37f", docs: "https://platform.openai.com/api-keys" },
  open_router:  { label: "OpenRouter",    color: "#ff6b35", docs: "https://openrouter.ai/keys" },
  openrouter:   { label: "OpenRouter",    color: "#ff6b35", docs: "https://openrouter.ai/keys" },
  gemini:       { label: "Gemini",        color: "#4285f4", docs: "https://aistudio.google.com/apikey" },
  anthropic:    { label: "Claude",        color: "#d97757", docs: "https://console.anthropic.com/settings/keys" },
  deepseek:     { label: "DeepSeek",      color: "#4f6ef7" },
  mistral:      { label: "Mistral",       color: "#ff6b6b" },
  groq:         { label: "Groq",          color: "#f97316" },
  cerebras:     { label: "Cerebras",      color: "#f59e0b" },
  nvidia_nim:   { label: "NVIDIA NIM",    color: "#76b900" },
  nvidia:       { label: "NVIDIA",        color: "#76b900" },
  ollama:       { label: "Ollama",        color: "#10b981" },
  lmstudio:     { label: "LM Studio",     color: "#6366f1" },
  llamacpp:     { label: "llama.cpp",     color: "#f43f5e" },
  "meta-llama":   { label: "Meta Llama",    color: "#2563eb" },
  "google":       { label: "Google",        color: "#4285f4" },
  "x-ai":         { label: "xAI Grok",      color: "#1da1f2" },
  "qwen":         { label: "Qwen",          color: "#06b6d4" },
  "cohere":       { label: "Cohere",        color: "#7c3aed" },
  "perplexity":   { label: "Perplexity",    color: "#64748b" },
  "mistralai":    { label: "Mistral AI",    color: "#ff6b6b" },
  "amazon":       { label: "Amazon",        color: "#ff9900" },
  "microsoft":    { label: "Microsoft",     color: "#00a4ef" },
  "ai21":         { label: "AI21 Labs",     color: "#6366f1" },
  "moonshotai":   { label: "Moonshot AI",   color: "#8b5cf6" },
  "minimax":      { label: "MiniMax",       color: "#06b6d4" },
  "tencent":      { label: "Tencent",       color: "#0052d9" },
  "baidu":        { label: "Baidu",         color: "#2932e1" },
  "bytedance":    { label: "ByteDance",     color: "#35c5f7" },
  "inflection":   { label: "Inflection",    color: "#f97316" },
  "writer":       { label: "Writer",        color: "#6366f1" },
  "upstage":      { label: "Upstage",       color: "#8b5cf6" },
  "poolside":     { label: "Poolside",      color: "#2563eb" },
  "rekaai":       { label: "Reka AI",       color: "#ec4899" },
  "nousresearch": { label: "Nous Research", color: "#a855f7" },
  "gryphe":       { label: "Gryphe",        color: "#f43f5e" },
  "undi95":       { label: "Undi95",        color: "#f97316" },
  "mancer":       { label: "Mancer",        color: "#64748b" },
  "sao10k":       { label: "Sao10k",        color: "#f59e0b" },
  "stepfun":      { label: "StepFun",       color: "#06b6d4" },
  "deepcogito":   { label: "DeepCogito",    color: "#4f6ef7" },
  "cognitivecomputations": { label: "Cognitive Computations", color: "#a855f7" },
  "essentialai":  { label: "Essential AI",  color: "#f97316" },
  "prime-intellect": { label: "Prime Intellect", color: "#8b5cf6" },
  "arcee-ai":     { label: "Arcee AI",      color: "#6366f1" },
  "aion-labs":    { label: "Aion Labs",     color: "#06b6d4" },
  "liquid":       { label: "Liquid",        color: "#ec4899" },
  "allenai":      { label: "Allen AI",      color: "#a855f7" },
  "morph":        { label: "Morph",         color: "#64748b" },
  "inclusionai":  { label: "Inclusion AI",  color: "#4f6ef7" },
  "perceptron":   { label: "Perceptron",    color: "#f59e0b" },
  "kwaipilot":    { label: "Kwaipilot",     color: "#2563eb" },
  "switchpoint":  { label: "Switchpoint",   color: "#f97316" },
  "thedrummer":   { label: "The Drummer",   color: "#8b5cf6" },
  "relace":       { label: "Relace",        color: "#06b6d4" },
  "nex-agi":      { label: "Nex AGI",       color: "#a855f7" },
  "xiaomi":       { label: "Xiaomi",        color: "#ff6b00" },
  "bytedance-seed": { label: "ByteDance Seed", color: "#35c5f7" },
  "z-ai":         { label: "Z.ai",          color: "#8b5cf6" },
  "ibm-granite":  { label: "IBM Granite",   color: "#0062ff" },
  "anthracite-org": { label: "Anthracite",  color: "#64748b" },
  "inception":    { label: "Inception",     color: "#ec4899" },
  "~anthropic": { label: "Anthropic",     color: "#d97757" },
  "~openai":    { label: "OpenAI",        color: "#10a37f" },
  "~google":    { label: "Google",        color: "#4285f4" },
  "~moonshotai":{ label: "Moonshot AI",   color: "#8b5cf6" },
  opencode:     { label: "OpenCode",      color: "#8b5cf6" },
};

const CORE_PROVIDERS = [
  { key: "OPENAI_API_KEY",    label: "OpenAI",     color: "#10a37f", docs: "https://platform.openai.com/api-keys" },
  { key: "ANTHROPIC_API_KEY", label: "Claude",     color: "#d97757", docs: "https://console.anthropic.com/settings/keys" },
  { key: "GEMINI_API_KEY",    label: "Gemini",     color: "#4285f4", docs: "https://aistudio.google.com/apikey" },
  { key: "OPENROUTER_API_KEY",label: "OpenRouter", color: "#ff6b35", docs: "https://openrouter.ai/keys" },
  { key: "OPENCODE_API_KEY",  label: "OpenCode",   color: "#8b5cf6", docs: "https://opencode.ai/keys" },
  { key: "DEEPSEEK_API_KEY",  label: "DeepSeek",   color: "#4f46e5", docs: "https://platform.deepseek.com/api-keys" },
  { key: "MISTRAL_API_KEY",   label: "Mistral",    color: "#f59e0b", docs: "https://console.mistral.ai/api-keys" },
  { key: "GROQ_API_KEY",      label: "Groq",       color: "#22c55e", docs: "https://console.groq.com/keys" },
];

const MODEL_TYPES = [
  { key: "MODEL",         label: "Default",      icon: "⚡", desc: "General purpose" },
  { key: "MODEL_CODE",    label: "Code",         icon: "💻", desc: "Coding, refactoring" },
  { key: "MODEL_IMAGE",   label: "Image Gen",    icon: "🎨", desc: "Images & editing" },
  { key: "MODEL_VIDEO",   label: "Video Gen",    icon: "🎬", desc: "Video generation" },
  { key: "MODEL_DOCUMENT",label: "Document Gen", icon: "📄", desc: "Writing, docs" },
];

function maskKey(value: string): string {
  if (value.length > 4) return "••••" + value.slice(-4);
  return value ? "••••" : "";
}

function getModelPrice(modelID: string): { input: number; output: number; context: number } | null {
  const parts = modelID.split("/");
  const name = parts.length > 1 ? parts.slice(1).join("/") : modelID;
  if (PRICING[name]) return PRICING[name];
  return null;
}

function getContextInfo(modelID: string): { size: number | null; label: string } {
  const price = getModelPrice(modelID);
  if (price) {
    const ctx = price.context;
    const label = ctx >= 1000000 ? `${(ctx / 1000000).toFixed(0)}M` : `${(ctx / 1000).toFixed(0)}K`;
    return { size: ctx, label };
  }
  return { size: null, label: "—" };
}

function modelNameOnly(modelID: string): string {
  const parts = modelID.split("/");
  return parts.length > 1 ? parts.slice(1).join("/") : modelID;
}

function providerID(modelID: string): string {
  const parts = modelID.split("/");
  return parts.length > 1 ? parts[0].toLowerCase() : "unknown";
}

function providerDisplay(id: string): { label: string; color: string } {
  return PROVIDER_DISPLAY[id] || { label: id, color: "var(--color-primary)" };
}

// ─── Model Selector Modal ──────────────────────────────────────────

function ModelSelector({ value, onChange, onClose, models, keyValues }: {
  value: string; onChange: (v: string) => void; onClose: () => void; models: ModelEntry[];
  keyValues?: Record<string, string>;
}) {
  const [query, setQuery] = useState("");
  const [tab, setTab] = useState<"remote" | "local">("remote");
  const ref = useRef<HTMLDivElement>(null);
  const [providerFilter, setProviderFilter] = useState<string>("all");
  const keyedProviders = useMemo(() => {
    const apiKeyMap: Record<string, string> = {
      OPENAI_API_KEY: "openai", ANTHROPIC_API_KEY: "anthropic", GEMINI_API_KEY: "gemini",
      OPENROUTER_API_KEY: "open_router", OPENCODE_API_KEY: "opencode",
      DEEPSEEK_API_KEY: "deepseek", MISTRAL_API_KEY: "mistral", GROQ_API_KEY: "groq",
      CEREBRAS_API_KEY: "cerebras", NVIDIA_NIM_API_KEY: "nvidia",
    };
    const configured = new Set<string>();
    if (keyValues) {
      for (const [envKey, providerId] of Object.entries(apiKeyMap)) {
        if (keyValues[envKey]?.length > 0) {
          configured.add(providerId);
          if (providerId === "open_router") {
            configured.add("anthropic");
            configured.add("openrouter");
          }
        }
      }
    }
    return configured;
  }, [keyValues]);

  const [localModels, setLocalModels] = useState<{ name: string; size?: number }[]>([]);
  const [loadingLocal, setLoadingLocal] = useState(false);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose();
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [onClose]);

  const fetchLocalModels = useCallback(async () => {
    setLoadingLocal(true);
    try {
      const r = await fetch("http://127.0.0.1:11434/api/tags", { signal: AbortSignal.timeout(3000) });
      if (r.ok) {
        const data = await r.json();
        setLocalModels((data.models || []).map((m: any) => ({ name: m.name, size: m.size })));
      }
    } catch {} finally { setLoadingLocal(false); }
  }, []);

  useEffect(() => { if (tab === "local") fetchLocalModels(); }, [tab, fetchLocalModels]);

  // Filter models to only show those from providers with configured API keys
  const filteredModels = keyedProviders.size > 0
    ? models.filter((m: any) => keyedProviders.has(m.provider || m.id.split("/")[0] || "other"))
    : models;

  const allProviders = [...new Set(filteredModels.map((m: any) => m.provider || m.id.split("/")[0] || "other"))].sort();

  const q = query.toLowerCase();
  const filteredByProvider = providerFilter === "all" ? filteredModels : filteredModels.filter((m: any) => (m.provider || m.id.split("/")[0] || "other") === providerFilter);
  const filteredRemote = q ? filteredByProvider.filter((m: any) => m.id.toLowerCase().includes(q)) : filteredByProvider;
  const filteredLocal = q ? localModels.filter((m) => m.name.toLowerCase().includes(q)) : localModels;

  // Group models by provider, with subgroup support (e.g. OpenCode Zen vs Go)
  const grouped: { provider: string; subgroups: { label: string; models: ModelEntry[] }[] }[] = [];
  const providerOrder: { [key: string]: { label: string; models: ModelEntry[] } } = {};

  for (const m of filteredRemote) {
    const prov = (m as any).provider || m.id.split("/")[0] || "other";
    if (!providerOrder[prov]) {
      providerOrder[prov] = { label: prov, models: [] };
    }
    providerOrder[prov].models.push(m);
  }

  for (const [prov, entry] of Object.entries(providerOrder)) {
    // Check if models have subgroups
    const hasSubgroups = entry.models.some((m: any) => m.subgroup);
    if (hasSubgroups) {
      const subgroups: { [key: string]: { label: string; models: ModelEntry[] } } = {};
      for (const m of entry.models) {
        const sg = (m as any).subgroup || "default";
        if (!subgroups[sg]) subgroups[sg] = { label: sg, models: [] };
        subgroups[sg].models.push(m);
      }
      grouped.push({
        provider: prov,
        subgroups: Object.entries(subgroups).map(([key, val]) => ({
          label: key,
          models: val.models,
        })),
      });
    } else {
      grouped.push({
        provider: prov,
        subgroups: [{ label: "", models: entry.models }],
      });
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-base-300/80 backdrop-blur-sm p-4">
      <div ref={ref} className="card bg-base-200 shadow-2xl max-w-lg w-full max-h-[80vh] overflow-y-auto">
        <div className="flex items-center justify-between px-5 py-4" style={{ borderBottom: "1px solid color-mix(in oklab, var(--color-base-content) 15%, transparent)" }}>
          <h3 className="text-sm font-semibold" style={{ color: "var(--color-base-content)" }}>Select Model</h3>
          <button className="btn btn-ghost btn-sm btn-square" onClick={onClose}><span className="material-symbols-outlined" style={{ fontSize: "20px" }}>close</span></button>
        </div>
        <div className="px-4 pt-3">
          <div className="relative">
            <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-base-content/60 text-sm">search</span>
            <input type="text" className="input input-bordered w-full pl-9" placeholder="Search models..." value={query} onChange={(e: any) => setQuery(e.target.value)} />
          </div>
        </div>
        {/* Provider filter pills */}
        <div className="flex gap-1 px-4 pt-3 overflow-x-auto">
          {[{ id: "all", label: `All (${filteredModels.length})` }, ...allProviders.slice(0, 15).map((p) => ({ id: p, label: `${p} (${filteredModels.filter((m) => (m.id.split("/")[0] || "other") === p).length})` }))].map((f) => (
            <button key={f.id} onClick={() => setProviderFilter(f.id)}
              className="px-2.5 py-1 rounded-lg text-[10px] font-medium transition whitespace-nowrap"
              style={{
                background: providerFilter === f.id ? "color-mix(in srgb, var(--color-primary) 15%, transparent)" : "var(--color-base-100)",
                color: providerFilter === f.id ? "var(--color-primary)" : "color-mix(in oklab, var(--color-base-content) 60%, transparent)",
              }}>
              {f.label}
            </button>
          ))}
        </div>
        <div className="flex gap-1 px-4 pt-2">
          <button onClick={() => setTab("remote")}
            className="px-3 py-1.5 rounded-lg text-[11px] font-medium transition"
            style={{
              background: tab === "remote" ? "color-mix(in srgb, var(--color-primary) 15%, transparent)" : "transparent",
              color: tab === "remote" ? "var(--color-primary)" : "color-mix(in oklab, var(--color-base-content) 60%, transparent)",
            }}>
            Models ({filteredRemote.length})
          </button>
          <button onClick={() => setTab("local")}
            className="px-3 py-1.5 rounded-lg text-[11px] font-medium transition flex items-center gap-1.5"
            style={{
              background: tab === "local" ? "color-mix(in srgb, var(--color-primary) 15%, transparent)" : "transparent",
              color: tab === "local" ? "var(--color-primary)" : "color-mix(in oklab, var(--color-base-content) 60%, transparent)",
            }}>
            <span className="material-symbols-outlined" style={{ fontSize: "14px" }}>memory</span> Local (Ollama)
          </button>
        </div>
        <div className="overflow-y-auto p-2" style={{ maxHeight: "400px" }}>
          {tab === "remote" && grouped.map((g) => {
            const pd = providerDisplay(g.provider);
            return (
              <div key={g.provider}>
                <div className="px-3 py-1.5 text-[10px] uppercase tracking-widest font-semibold sticky top-0 flex items-center gap-2"
                  style={{ color: pd.color, background: "var(--color-base-200)", borderBottom: `1px solid ${pd.color}20` }}>
                  {pd.label}
                </div>
                {g.subgroups.map((sg) => (
                  <div key={sg.label || "__main"}>
                    {sg.label && (
                      <div className="px-3 py-1 text-[9px] uppercase tracking-widest font-medium flex items-center gap-1.5"
                        style={{ color: `${pd.color}99` }}>
                        <span style={{ width: 8, height: 1, background: `${pd.color}40`, display: "inline-block" }} />
                        {sg.label === "zen" ? "Zen (Cloud API)" : sg.label === "go" ? "Go (Cloud API)" : sg.label}
                      </div>
                    )}
                    {sg.models.map((m) => {
                      const selected = m.id === value;
                      const price = getModelPrice(m.id);
                      const ctx = getContextInfo(m.id);
                      const isFree = m.id.includes(":free") || m.id.endsWith("-free");
                      return (
                        <button key={m.id} onClick={() => { onChange(m.id); onClose(); }}
                          className="w-full text-left px-3 py-2.5 rounded-lg text-[12px] transition flex items-center justify-between"
                          style={{ background: selected ? `${pd.color}18` : "transparent", color: selected ? pd.color : "var(--color-base-content)" }}>
                          <div className="min-w-0 flex-1">
                            <div className="flex items-center gap-2">
                              <span className="truncate font-mono">{modelNameOnly(m.id)}</span>
                              <span className="text-[9px] px-1.5 py-0.5 rounded uppercase tracking-wider shrink-0" style={{ background: `${pd.color}20`, color: pd.color }}>{pd.label}</span>
                              {sg.label && <span className="text-[9px] px-1.5 py-0.5 rounded uppercase tracking-wider shrink-0" style={{ background: `${pd.color}10`, color: `${pd.color}cc` }}>{sg.label === "zen" ? "Zen" : sg.label === "go" ? "Go" : sg.label}</span>}
                              {isFree && <span className="text-[9px] px-1.5 py-0.5 rounded uppercase tracking-wider shrink-0" style={{ background: "color-mix(in srgb, var(--color-tertiary) 12%, transparent)", color: "var(--color-tertiary)" }}>Free</span>}
                              {!isFree && price && <span className="text-[9px] px-1.5 py-0.5 rounded uppercase tracking-wider shrink-0" style={{ background: "color-mix(in srgb, var(--color-error) 12%, transparent)", color: "var(--color-error)" }}>$</span>}
                            </div>
                            <div className="flex items-center gap-2 mt-0.5 flex-wrap">
                              {price && (<>
                                <span className="text-[10px] font-mono" style={{ color: "color-mix(in oklab, var(--color-base-content) 60%, transparent)" }}>ctx: {ctx.label}</span>
                                <span className="text-[10px] font-mono" style={{ color: "color-mix(in oklab, var(--color-base-content) 60%, transparent)" }}>in: ${price.input.toFixed(2)}/M</span>
                                <span className="text-[10px] font-mono" style={{ color: "color-mix(in oklab, var(--color-base-content) 60%, transparent)" }}>out: ${price.output.toFixed(2)}/M</span>
                                <span className="text-[10px]" style={{ color: "color-mix(in oklab, var(--color-base-content) 60%, transparent)" }}>
                                  ~${((price.input * 0.0005 + price.output * 0.0002)).toFixed(4)}/chat
                                </span>
                              </>)}
                              {!price && !isFree && <span className="text-[10px]" style={{ color: "color-mix(in oklab, var(--color-base-content) 60%, transparent)" }}>ctx: {ctx.label}</span>}
                              {isFree && <span className="text-[10px]" style={{ color: "var(--color-tertiary)" }}>No cost — free tier</span>}
                            </div>
                          </div>
                          {selected && <span className="material-symbols-outlined" style={{ fontSize: "16px", color: pd.color }}>check_circle</span>}
                        </button>
                      );
                    })}
                  </div>
                ))}
              </div>
            );
          })}
          {tab === "remote" && models.length === 0 && (
            <div className="p-4 text-center text-[12px]" style={{ color: "color-mix(in oklab, var(--color-base-content) 60%, transparent)" }}>
              {query ? "No models match your search" : "Loading models…"}
            </div>
          )}
          {tab === "local" && (
            <>
              {loadingLocal && (<div className="flex items-center justify-center gap-2 p-4 text-[12px]" style={{ color: "color-mix(in oklab, var(--color-base-content) 60%, transparent)" }}><span className="loading loading-spinner loading-sm"></span> Loading local models...</div>)}
              {!loadingLocal && filteredLocal.length === 0 && (<div className="p-4 text-center text-[12px]" style={{ color: "color-mix(in oklab, var(--color-base-content) 60%, transparent)" }}>{query ? "No models match" : "No models in Ollama. Pull one first."}</div>)}
              {!loadingLocal && filteredLocal.map((m) => {
                const fullName = "ollama/" + m.name;
                const selected = fullName === value;
                return (
                  <button key={m.name} onClick={() => { onChange(fullName); onClose(); }}
                    className="w-full text-left px-3 py-2.5 rounded-lg text-[12px] font-mono transition flex items-center justify-between"
                    style={{ background: selected ? "color-mix(in srgb, var(--color-primary) 15%, transparent)" : "transparent", color: selected ? "var(--color-primary)" : "var(--color-base-content)" }}>
                    <div className="min-w-0 flex-1">
                      <div className="truncate">{m.name}</div>
                      <div className="flex items-center gap-3 mt-0.5">
                        {m.size && <span className="text-[10px]" style={{ color: "color-mix(in oklab, var(--color-base-content) 60%, transparent)" }}>{(m.size / 1e9).toFixed(1)}GB</span>}
                        <span className="text-[10px]" style={{ color: "color-mix(in oklab, var(--color-base-content) 60%, transparent)" }}>local · free</span>
                      </div>
                    </div>
                    {selected && <span className="material-symbols-outlined" style={{ fontSize: "16px" }}>check_circle</span>}
                  </button>
                );
              })}
            </>
          )}
        </div>
        <div className="px-4 py-3 text-[10px] leading-relaxed" style={{ borderTop: "1px solid color-mix(in oklab, var(--color-base-content) 15%, transparent)", color: "color-mix(in oklab, var(--color-base-content) 60%, transparent)" }}>
          Format: <strong>provider</strong>/<strong>model</strong> — e.g. <code className="text-[10px]">openai/gpt-4o</code>, <code className="text-[10px]">ollama/llama3.1</code>
        </div>
      </div>
    </div>
  );
}

// ─── Main Config Component ─────────────────────────────────────────

export default function TalonConfig() {
  const [settings, setSettings] = useState<EchoSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: "ok" | "err"; text: string } | null>(null);
  const [edits, setEdits] = useState<Record<string, string>>({});
  const [models, setModels] = useState<ModelEntry[]>([]);
  const [echoReachable, setEchoReachable] = useState<boolean | null>(null);
  const [modelsLoading, setModelsLoading] = useState(false);
  const [visibleKeys, setVisibleKeys] = useState<Set<string>>(new Set());
  const [selectingModel, setSelectingModel] = useState<string | null>(null);
  const [useDefaultForAll, setUseDefaultForAll] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const r = await fetch("/api/talon/admin/config");
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        const cfgData = await r.json();
        if (cancelled) return;
        setSettings(cfgData);
        const merged: Record<string, string> = {};
        for (const p of CORE_PROVIDERS) merged[p.key] = cfgData.config?.[p.key] ?? "";
        for (const m of MODEL_TYPES) merged[m.key] = cfgData.config?.[m.key] ?? "";
        merged["OLLAMA_BASE_URL"] = cfgData.config?.["OLLAMA_BASE_URL"] ?? "http://localhost:11434";
        setEdits(merged);
        try {
          const hr = await fetch("/api/talon/health", { signal: AbortSignal.timeout(1500) });
          const data = await hr.json();
          if (!cancelled) setEchoReachable(data.status === "healthy");
        } catch { if (!cancelled) setEchoReachable(false); }
      } catch { if (!cancelled) setMessage({ type: "err", text: "Failed to load config" }); }
      finally { if (!cancelled) setLoading(false); }
    })();
    return () => { cancelled = true; };
  }, []);

  const refreshModels = useCallback(async () => {
    setModelsLoading(true);
    try {
      const r = await fetch("/api/talon/models", { signal: AbortSignal.timeout(5000) });
      if (r.ok) {
        const data = await r.json();
        const entries: ModelEntry[] = Array.isArray(data?.data) ? data.data : [];
        entries.sort((a: ModelEntry, b: ModelEntry) => {
          const aProv = (a as any).provider || a.id.split("/")[0] || "";
          const bProv = (b as any).provider || b.id.split("/")[0] || "";
          return aProv.localeCompare(bProv) || a.id.localeCompare(b.id);
        });
        setModels(entries);
      }
    } catch {} finally { setModelsLoading(false); }
  }, []);

  useEffect(() => { if (echoReachable) refreshModels(); }, [echoReachable, refreshModels]);

  async function save() {
    setSaving(true); setMessage(null);
    try {
      const r = await fetch("/api/talon/admin/config", {
        method: "PUT", headers: { "content-type": "application/json" },
        body: JSON.stringify(edits),
      });
      const data = await r.json();
      if (r.ok) {
        setMessage({ type: "ok", text: `Saved — ${data.updated} values updated.` });
        const fresh = await fetch("/api/talon/admin/config").then((res) => res.json());
        setSettings(fresh);
      } else setMessage({ type: "err", text: data.error || "Save failed" });
    } catch { setMessage({ type: "err", text: "Network error" }); }
    setSaving(false);
  }

  function setKey(key: string, value: string) {
    setEdits((prev) => ({ ...prev, [key]: value }));
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center" style={{ height: "192px" }}>
        <span className="loading loading-spinner loading-sm text-primary"></span>
      </div>
    );
  }

  const configuredCount = CORE_PROVIDERS.filter((p) => (edits[p.key] ?? "").length > 0).length;

  return (
    <div className="space-y-5">
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-semibold" style={{ fontFamily: "'Roboto', sans-serif" }}>
              Talon Configuration
            </h2>
            <div
              className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[10px] font-medium uppercase tracking-widest ${
                echoReachable === true
                  ? "bg-success/10 text-success"
                  : echoReachable === false
                    ? "bg-error/10 text-error"
                    : "bg-base-content/10 text-base-content/60"
              }`}
            >
              {echoReachable === true ? <><span className="material-symbols-outlined" style={{ fontSize: "12px" }}>check_circle</span> Proxy Active</> : null}
              {echoReachable === false ? <><span className="material-symbols-outlined" style={{ fontSize: "12px" }}>error</span> Proxy Offline</> : null}
              {echoReachable === null ? <><span className="material-symbols-outlined" style={{ fontSize: "12px" }}>pending</span> Checking…</> : null}
            </div>
          </div>
          <p className="text-xs mt-1 text-base-content/60">
            Settings in <code className="px-1.5 py-0.5 rounded text-[11px] bg-base-200">~/.talon/.env</code>
          </p>
        </div>
        <button className="btn btn-primary" onClick={save} disabled={saving}>
          <span className="material-symbols-outlined" style={{ fontSize: "16px" }}>save</span>
          {saving ? "Saving…" : "Save Settings"}
        </button>
      </div>

      {message && (
        <div className={`flex items-center gap-2 px-4 py-3 rounded-xl ${message.type === "ok" ? "bg-success/10 text-success" : "bg-error/10 text-error"}`}>
          <span className="material-symbols-outlined" style={{ fontSize: "18px" }}>{message.type === "ok" ? "check_circle" : "error"}</span>
          {message.text}
        </div>
      )}

      {/* Provider Keys */}
      <div className="rounded-xl bg-base-200 border border-base-content/10 p-5 shadow-sm">
        <div className="flex items-center gap-2 mb-4">
          <span className="material-symbols-outlined text-primary" style={{ fontSize: "18px" }}>key</span>
          <h3 className="font-bold text-sm">API Providers</h3>
          <span className="badge badge-sm">{configuredCount} configured</span>
        </div>
        <div className="flex flex-col gap-2">
          {CORE_PROVIDERS.map((p) => {
            const val = edits[p.key] ?? "";
            const isSet = val.length > 0;
            const visible = visibleKeys.has(p.key);
            return (
              <div key={p.key} className="rounded-lg bg-base-300 border border-base-content/5 px-3 py-2.5 flex items-center gap-3 transition flex-wrap">
                <div className="w-2.5 h-2.5 rounded-full shrink-0" style={{ background: p.color }} />
                <label className={`w-20 shrink-0 text-sm${!isSet ? " text-base-content/60" : ""}`} style={isSet ? { color: p.color } : undefined}>{p.label}</label>
                <div className="flex-1 flex items-center gap-2 min-w-[200px]">
                  <div className="relative w-full">
                    <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-base-content/60 text-sm">key</span>
                    <input
                      type={visible ? "text" : "password"}
                      className="input input-bordered bg-base-300 border-base-content/20 w-full pl-9"
                      placeholder={`${p.label} API key…`}
                      value={val}
                      onChange={(e: any) => setKey(p.key, e.target.value)}
                    />
                  </div>
                  <button className="btn btn-ghost btn-xs btn-square" onClick={() => setVisibleKeys((prev) => { const n = new Set(prev); n.has(p.key) ? n.delete(p.key) : n.add(p.key); return n; })}>
                    <span className="material-symbols-outlined" style={{ fontSize: "18px" }}>{visible ? "visibility_off" : "visibility"}</span>
                  </button>
                </div>
                {p.docs && (
                  <a href={p.docs} target="_blank" rel="noopener noreferrer"
                    className="btn btn-ghost btn-sm shrink-0 flex items-center gap-1">
                    <span className="material-symbols-outlined" style={{ fontSize: "14px" }}>open_in_new</span>Get
                  </a>
                )}
              </div>
            );
          })}
        </div>
      </div>

      {/* Model Routing */}
      <div className="rounded-xl bg-base-200 border border-base-content/10 p-5 shadow-sm">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <span className="material-symbols-outlined text-primary" style={{ fontSize: "18px" }}>smart_toy</span>
            <h3 className="font-bold text-sm">Model Routing</h3>
          </div>
          <div className="flex items-center gap-2">
            {modelsLoading && <span className="loading loading-spinner loading-sm"></span>}
            <button className="btn btn-ghost btn-sm" onClick={refreshModels}>
              <span className="material-symbols-outlined" style={{ fontSize: "16px" }}>refresh</span>
              Refresh
            </button>
            {models.length > 0 && <span className="text-xs text-base-content/60">{models.length} models</span>}
          </div>
        </div>
        <div className="rounded-lg bg-base-300 border border-base-content/5 px-3 py-2.5 flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <span className="text-sm">Use Default for all tasks</span>
            <span className="text-xs text-base-content/60">— single model for everything</span>
          </div>
          <input type="checkbox" className="toggle toggle-sm" checked={useDefaultForAll} onChange={() => setUseDefaultForAll(!useDefaultForAll)} />
        </div>
        <div className="space-y-2">
          {MODEL_TYPES.map((mt) => {
            if (useDefaultForAll && mt.key !== "MODEL") return null;
            const val = edits[mt.key] ?? "";
            const isSet = val.length > 0;
            const price = getModelPrice(val);
            const ctx = getContextInfo(val);
            const pd = providerDisplay(providerID(val));
            return (
              <div key={mt.key} className="rounded-lg bg-base-300 border border-base-content/5 px-3 py-2.5 flex items-center gap-3 transition cursor-pointer"
                onClick={() => setSelectingModel(mt.key)}>
                <span className="text-lg shrink-0">{mt.icon}</span>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium">{mt.label}</span>
                    <span className="text-xs text-base-content/60">{mt.desc}</span>
                    {isSet && (<span className="badge badge-sm" style={{ background: `${pd.color}20`, color: pd.color }}>{pd.label}</span>)}
                  </div>
                  <div className="flex items-center gap-3 mt-0.5">
                    <span className={`text-xs font-mono truncate ${isSet ? "" : "text-base-content/60"}`} style={isSet ? { color: pd.color } : undefined}>
                      {isSet ? modelNameOnly(val) : "Select a model…"}
                    </span>
                    {isSet && (<><span className="text-[10px] text-base-content/60">ctx: {ctx.label}</span>{price && <span className="text-[10px] text-base-content/60">${price.input.toFixed(2)}/${price.output.toFixed(2)}</span>}</>)}
                  </div>
                </div>
                <span className="material-symbols-outlined text-base-content/60" style={{ fontSize: "16px" }}>expand_more</span>
              </div>
            );
          })}
        </div>
      </div>

      {selectingModel && (
        <ModelSelector value={edits[selectingModel] ?? ""}
          onChange={(v) => { setKey(selectingModel, v); setSelectingModel(null); }}
          keyValues={{ ...edits, ...(settings?.config ?? {}) }}
          onClose={() => setSelectingModel(null)} models={models} />
      )}

      {/* Pricing Info */}
      <details className="rounded-xl bg-base-200 border border-base-content/10 p-5 shadow-sm">
        <summary className="flex items-center gap-2 cursor-pointer text-sm font-bold list-none">
          <span className="material-symbols-outlined text-primary" style={{ fontSize: "18px" }}>info</span>
          How pricing and tokens work
          <span className="material-symbols-outlined ml-auto text-base-content/60" style={{ fontSize: "16px" }}>expand_more</span>
        </summary>
        <div className="mt-4 space-y-3 text-xs text-base-content/60 leading-relaxed">
          <p>Tokens are how AI models measure text. A token is roughly ¾ of a word.</p>
          <p>Pricing is shown per 1 million tokens. Input is what you send, output is what the model generates.</p>
          <p>Local models (Ollama) cost nothing but use your hardware.</p>
        </div>
      </details>

      <div className="flex gap-3 flex-wrap pb-4">
        <span className="rounded-full border border-base-content/10 px-3 py-1.5 text-[10px] text-base-content/60">
          Config: ~/.talon/.env
        </span>
        {echoReachable && models.length > 0 && (
          <span className="rounded-full border border-primary/20 px-3 py-1.5 text-[10px] text-primary">
            {models.length} models from {new Set(models.map(m => m.provider)).size} providers
          </span>
        )}
      </div>
    </div>
  );
}
