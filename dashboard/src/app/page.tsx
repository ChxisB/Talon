"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import {
  Activity, Image, ListChecks, Clock, ArrowRight, Wrench,
  Cpu, CheckCircle2, Network, Coins, MessageSquare, Layers, Database
} from "lucide-react";

interface AgentStatus { status: string; model: string | null; provider: string | null; latency: number | null; }
interface TaskStats { total: number; pending: number; running: number; completed: number; failed: number; }
interface ActivityEntry { ts: number; type: string; agent: string; text: string; }
interface TotalStats {
  total_sessions: number;
  total_prompt_tokens: number;
  total_completion_tokens: number;
  total_cost: number;
  total_messages: number;
  avg_tokens_per_session: number;
  avg_messages_per_session: number;
}
interface DailyUsage {
  day: string; prompt_tokens: number; completion_tokens: number; cost: number; session_count: number;
}
interface ModelUsage {
  model: string; provider: string; message_count: number;
}

function StatCard({ label, value, color, icon }: { label: string; value: string | number; color: string; icon: React.ReactNode }) {
  return (
    <div className="rounded-xl bg-base-200 border border-base-content/10 p-4 shadow-sm">
      <div className="flex items-center gap-3 mb-2">
        <span className={`${color} shrink-0`}>{icon}</span>
        <span className="text-xs font-semibold uppercase tracking-wider text-base-content/70">{label}</span>
      </div>
      <div className={`text-2xl font-bold ${color}`}>{value}</div>
    </div>
  );
}

function fmtTokens(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + "M";
  if (n >= 1_000) return (n / 1_000).toFixed(1) + "K";
  return String(n);
}

export default function Dashboard() {
  const [agent, setAgent] = useState<AgentStatus>({ status: "loading", model: null, provider: null, latency: null });
  const [stats, setStats] = useState<TaskStats | null>(null);
  const [activity, setActivity] = useState<ActivityEntry[]>([]);
  const [tokenStats, setTokenStats] = useState<TotalStats | null>(null);
  const [daily, setDaily] = useState<DailyUsage[]>([]);
  const [models, setModels] = useState<ModelUsage[]>([]);
  const [tokenStatsError, setTokenStatsError] = useState(false);

  useEffect(() => {
    Promise.all([
      fetch("/api/talon/status").then(r => r.json()).catch(() => ({ status: "error" })),
      fetch("/api/talon/admin/config").then(r => r.json()).catch(() => ({ config: {} })),
      fetch("/api/talon/tasks").then(r => r.json()).catch(() => ({ stats: null })),
      fetch("/api/talon/activity", { cache: "no-store" }).then(r => r.json()).catch(() => ({ entries: [] })),
      fetch("/api/talon/stats", { cache: "no-store" }).then(r => r.json()).catch(() => null),
    ]).then(([status, config, tasks, act, ts]) => {
      setAgent({
        status: status.status || "error",
        model: config.config?.MODEL || null,
        provider: config.config?.MODEL?.split("/")[0] || null,
        latency: status.latency || null,
      });
      if (tasks.stats) setStats(tasks.stats);
      setActivity(act.entries?.slice(0, 5) || []);
      if (ts && ts.total) {
        setTokenStats(ts.total);
        setDaily(ts.daily || []);
        setModels(ts.models || []);
      } else {
        setTokenStatsError(true);
      }
    });
  }, []);

  const healthy = agent.status === "ok" || agent.status === "healthy";

  return (
    <div className="flex flex-col gap-6">

      {/* Agent Status Banner */}
      <div className={`rounded-xl border p-5 shadow-sm flex items-center justify-between flex-wrap gap-4 ${
        healthy ? "bg-base-200 border-base-content/10" : "bg-error/10 border-error/30"
      }`}>
        <div className="flex items-center gap-4">
          <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${healthy ? "bg-primary/15" : "bg-error/20"}`}>
            <Cpu size={24} className={healthy ? "text-primary" : "text-error"} />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className={`w-2.5 h-2.5 rounded-full ${healthy ? "bg-success" : "bg-error"}`} />
              <span className="font-bold text-lg">{healthy ? "Agent Online" : "Agent Offline"}</span>
            </div>
            <div className="flex items-center gap-3 mt-1 text-sm text-base-content/60 flex-wrap">
              {agent.model && <span className="font-mono text-xs">{agent.model}</span>}
              {agent.provider && <span className="badge badge-ghost badge-xs">{agent.provider}</span>}
              {agent.latency !== null && <span className="text-xs">{agent.latency}ms latency</span>}
            </div>
          </div>
        </div>
        <Link href="/tools" className="btn btn-primary btn-sm gap-2">
          <Wrench size={14} /> Configure
        </Link>
      </div>

      {/* Task Stats */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <StatCard label="Running" value={stats?.running ?? "—"} color="text-info" icon={<Activity size={16} />} />
        <StatCard label="Completed" value={stats?.completed ?? "—"} color="text-success" icon={<CheckCircle2 size={16} />} />
        <StatCard label="Failed" value={stats?.failed ?? "—"} color="text-error" icon={<Activity size={16} />} />
        <StatCard label="Total Tasks" value={stats?.total ?? 0} color="text-base-content" icon={<ListChecks size={16} />} />
      </div>

      {/* Token Usage Stats */}
      {tokenStats && (
        <>
          <div>
            <div className="flex items-center gap-2 mb-3">
              <div className="w-1 h-5 rounded-full bg-secondary" />
              <Coins size={14} className="text-secondary" />
              <h2 className="font-bold text-sm tracking-tight">Token Usage</h2>
            </div>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              <StatCard label="Sessions" value={tokenStats.total_sessions} color="text-base-content" icon={<Layers size={16} />} />
              <StatCard label="Prompt Tokens" value={fmtTokens(tokenStats.total_prompt_tokens)} color="text-secondary" icon={<MessageSquare size={16} />} />
              <StatCard label="Completion Tokens" value={fmtTokens(tokenStats.total_completion_tokens)} color="text-accent" icon={<MessageSquare size={16} />} />
              <StatCard label="Total Cost" value={`$${tokenStats.total_cost.toFixed(4)}`} color="text-base-content" icon={<Coins size={16} />} />
            </div>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-3">
              <StatCard label="Total Messages" value={tokenStats.total_messages} color="text-base-content" icon={<Database size={16} />} />
              <StatCard label="Avg Tokens / Session" value={fmtTokens(Math.round(tokenStats.avg_tokens_per_session))} color="text-base-content/70" icon={<Activity size={16} />} />
              <StatCard label="Avg Msgs / Session" value={Math.round(tokenStats.avg_messages_per_session)} color="text-base-content/70" icon={<Activity size={16} />} />
              <StatCard label="Model" value={models[0]?.model || "—"} color="text-base-content/70" icon={<Cpu size={16} />} />
            </div>
          </div>
        </>
      )}

      {tokenStatsError && !tokenStats && (
        <div className="rounded-xl bg-warning/10 border border-warning/30 p-4 text-sm text-warning">
          Token stats unavailable — talon.db not found. Start a session or set TALON_DATA_DIR.
        </div>
      )}

      {/* Daily Activity */}
      {daily.length > 0 && (
        <div className="rounded-xl bg-base-200 border border-base-content/10 p-5 shadow-sm">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-1 h-6 rounded-full bg-primary" />
            <div className="flex items-center gap-2 flex-1">
              <Clock size={16} className="text-base-content/70" />
              <h2 className="font-bold text-sm tracking-tight">Daily Activity (last 30 days)</h2>
            </div>
          </div>
          <div className="overflow-x-auto">
            <table className="table table-xs w-full">
              <thead>
                <tr>
                  <th>Day</th>
                  <th>Sessions</th>
                  <th>Prompt</th>
                  <th>Completion</th>
                  <th>Cost</th>
                </tr>
              </thead>
              <tbody>
                {daily.slice(0, 14).map(d => (
                  <tr key={d.day}>
                    <td className="font-mono">{d.day}</td>
                    <td>{d.session_count}</td>
                    <td className="font-mono text-xs">{fmtTokens(d.prompt_tokens)}</td>
                    <td className="font-mono text-xs">{fmtTokens(d.completion_tokens)}</td>
                    <td className="font-mono text-xs">${d.cost.toFixed(4)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Model Breakdown */}
      {models.length > 0 && (
        <div className="rounded-xl bg-base-200 border border-base-content/10 p-5 shadow-sm">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-1 h-6 rounded-full bg-accent" />
            <div className="flex items-center gap-2 flex-1">
              <Cpu size={16} className="text-base-content/70" />
              <h2 className="font-bold text-sm tracking-tight">Model Usage</h2>
            </div>
          </div>
          <div className="overflow-x-auto">
            <table className="table table-xs w-full">
              <thead>
                <tr>
                  <th>Model</th>
                  <th>Provider</th>
                  <th>Messages</th>
                </tr>
              </thead>
              <tbody>
                {models.map(m => (
                  <tr key={`${m.provider}/${m.model}`}>
                    <td className="font-mono text-xs">{m.model}</td>
                    <td><span className="badge badge-ghost badge-xs">{m.provider}</span></td>
                    <td>{m.message_count}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Quick Navigation */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <Link href="/kanban" className="rounded-xl bg-base-200 border border-base-content/10 p-5 hover:border-primary/30 transition-all shadow-sm group">
          <div className="flex items-center gap-3 mb-3">
            <div className="w-10 h-10 rounded-xl bg-primary/15 flex items-center justify-center">
              <ListChecks size={20} className="text-primary" />
            </div>
            <div>
              <div className="font-bold text-sm">Tasks</div>
              <div className="text-xs text-base-content/70">Full task board & kanban</div>
            </div>
          </div>
          <div className="text-xs text-base-content/60 flex items-center gap-1 group-hover:text-primary transition-colors">
            Open tasks <ArrowRight size={12} />
          </div>
        </Link>

        <Link href="/memory" className="rounded-xl bg-base-200 border border-base-content/10 p-5 hover:border-primary/30 transition-all shadow-sm group">
          <div className="flex items-center gap-3 mb-3">
            <div className="w-10 h-10 rounded-xl bg-primary/15 flex items-center justify-center">
              <Network size={20} className="text-primary" />
            </div>
            <div>
              <div className="font-bold text-sm">Code Graph</div>
              <div className="text-xs text-base-content/70">Analyse codebase structure</div>
            </div>
          </div>
          <div className="text-xs text-base-content/60 flex items-center gap-1 group-hover:text-primary transition-colors">
            Explore graph <ArrowRight size={12} />
          </div>
        </Link>

        <Link href="/diagrams" className="rounded-xl bg-base-200 border border-base-content/10 p-5 hover:border-primary/30 transition-all shadow-sm group">
          <div className="flex items-center gap-3 mb-3">
            <div className="w-10 h-10 rounded-xl bg-primary/15 flex items-center justify-center">
              <Image size={20} className="text-primary" />
            </div>
            <div>
              <div className="font-bold text-sm">Diagrams</div>
              <div className="text-xs text-base-content/70">Architecture, flowcharts & more</div>
            </div>
          </div>
          <div className="text-xs text-base-content/60 flex items-center gap-1 group-hover:text-primary transition-colors">
            Build diagrams <ArrowRight size={12} />
          </div>
        </Link>

        <Link href="/tools" className="rounded-xl bg-base-200 border border-base-content/10 p-5 hover:border-primary/30 transition-all shadow-sm group">
          <div className="flex items-center gap-3 mb-3">
            <div className="w-10 h-10 rounded-xl bg-primary/15 flex items-center justify-center">
              <Wrench size={20} className="text-primary" />
            </div>
            <div>
              <div className="font-bold text-sm">Tools</div>
              <div className="text-xs text-base-content/70">Config, plugins, cron</div>
            </div>
          </div>
          <div className="text-xs text-base-content/60 flex items-center gap-1 group-hover:text-primary transition-colors">
            Manage tools <ArrowRight size={12} />
          </div>
        </Link>
      </div>

      {/* Activity */}
      {activity.length > 0 && (
        <div className="rounded-xl bg-base-200 border border-base-content/10 p-5 shadow-sm">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-1 h-6 rounded-full bg-primary" />
            <div className="flex items-center gap-2 flex-1">
              <Clock size={16} className="text-base-content/70" />
              <h2 className="font-bold text-sm tracking-tight">Recent Activity</h2>
            </div>
          </div>
          <div className="flex flex-col gap-1.5 max-h-48 overflow-y-auto">
            {activity.map((e, i) => {
              const isErr = e.text.toLowerCase().includes("error") || e.text.toLowerCase().includes("fail");
              return (
                <div key={`${e.ts}-${i}`} className="flex items-start gap-3 px-3 py-2 rounded-lg bg-base-300 border border-base-content/5">
                  <span className={`w-2 h-2 rounded-full mt-1.5 shrink-0 ${isErr ? "bg-error" : "bg-primary"}`} />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 text-xs">
                      <span className={`font-bold uppercase tracking-wider ${isErr ? "text-error" : "text-primary"}`}>{e.type}</span>
                      <span className="text-base-content/70">·</span>
                      <span className="text-base-content/70">{e.agent}</span>
                      <span className="text-base-content/60 ml-auto">{new Date(e.ts).toLocaleTimeString()}</span>
                    </div>
                    <p className={`text-sm mt-0.5 leading-relaxed ${isErr ? "text-error" : "text-base-content/70"}`}>{e.text}</p>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
