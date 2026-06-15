"use client";

import { useState, useEffect } from "react";

interface ActivityEntry { ts: number; type: string; agent: string; text: string; }

const TYPE_META: Record<string, { icon: string; color: string; label: string }> = {
  task:    { icon: "smart_toy",     color: "text-primary",    label: "Task" },
  result:  { icon: "check_circle",  color: "text-success",    label: "Result" },
  cron:    { icon: "schedule",      color: "text-info",       label: "Cron" },
  error:   { icon: "error",         color: "text-error",      label: "Error" },
  dream:   { icon: "auto_awesome",  color: "text-secondary",  label: "Dream" },
};

export default function ActivityPage() {
  const [entries, setEntries] = useState<ActivityEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState("all");

  useEffect(() => {
    let stop = false;
    const fetchIt = async () => {
      try {
        const r = await fetch("/api/talon/activity", { cache: "no-store" });
        const j = await r.json();
        if (!stop) setEntries(j.entries ?? []);
      } catch {}
      if (!stop) setLoading(false);
    };
    fetchIt();
    const t = setInterval(fetchIt, 8000);
    return () => { stop = true; clearInterval(t); };
  }, []);

  const filtered = filter === "all" ? entries : entries.filter(e => e.type === filter);

  if (loading) {
    return <div className="flex items-center justify-center py-16"><span className="loading loading-spinner loading-lg" /></div>;
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Filter */}
      <div className="flex flex-wrap gap-2">
        {[{ id: "all", label: "All" }, ...Object.entries(TYPE_META).map(([id, m]) => ({ id, label: m.label }))].map(f => (
          <button key={f.id} className={`badge badge-outline py-3 cursor-pointer ${filter === f.id ? "badge-primary" : ""}`} onClick={() => setFilter(f.id)}>
            {f.label}
          </button>
        ))}
      </div>

      {/* List */}
      <div className="flex flex-col gap-2">
        {filtered.map((e, i) => {
          const meta = TYPE_META[e.type] || { icon: "timeline", color: "text-base-content/60", label: e.type };
          const time = new Date(e.ts).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", second: "2-digit" });
          const isErr = e.text.toLowerCase().includes("error") || e.text.toLowerCase().includes("fail");
          return (
            <div key={`${e.ts}-${i}`} className="card card-bordered bg-base-100 border-base-300">
              <div className="card-body p-4">
                <div className="flex items-start gap-3">
                  <div className={`w-9 h-9 rounded-xl flex items-center justify-center shrink-0 ${meta.color} bg-primary/5`}>
                    <span className={`material-symbols-outlined text-sm ${meta.color}`}>{meta.icon}</span>
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className={`font-semibold text-xs ${isErr ? "text-error" : meta.color}`}>{meta.label}</span>
                      <div className="badge badge-ghost badge-sm">{e.agent}</div>
                      <span className="text-xs font-mono text-base-content/60 ml-auto">{time}</span>
                    </div>
                    <p className={`text-sm mt-1 leading-relaxed ${isErr ? "text-error" : "text-base-content/60"}`}>{e.text}</p>
                  </div>
                </div>
              </div>
            </div>
          );
        })}
        {filtered.length === 0 && (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <span className="material-symbols-outlined text-4xl text-base-content/70 mb-3">notifications_off</span>
            <p className="text-base-content/60">No activity yet.</p>
            <p className="text-sm text-base-content/60 mt-1">Tasks and cron jobs will appear here.</p>
          </div>
        )}
      </div>
    </div>
  );
}
