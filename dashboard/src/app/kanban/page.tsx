"use client";

import { useState, useEffect } from "react";
import { ListChecks, Activity } from "lucide-react";

interface BackendTask { id: string; description: string; status: string; progress: number; created_at: string; }
interface TaskStats { total: number; pending: number; running: number; completed: number; failed: number; }
interface LocalTask { id: string; text: string; column: "todo" | "progress" | "done"; }

const STORAGE_KEY = "talon-kanban";
const COLUMNS = [
  { id: "todo" as const,     label: "To Do",         icon: "radio_button_unchecked", color: "text-base-content/70" },
  { id: "progress" as const, label: "In Progress",    icon: "pending",               color: "text-info" },
  { id: "done" as const,     label: "Done",           icon: "check_circle",          color: "text-success" },
];

const statusIcon = (s: string) => {
  switch(s) { case "completed": return "check_circle"; case "failed": return "error"; case "running": return "sync"; default: return "schedule"; }
};
const statusColor = (s: string) => {
  switch(s) { case "completed": return "text-success"; case "failed": return "text-error"; case "running": return "text-info"; default: return "text-base-content/70"; }
};
const statusBadge = (s: string) => {
  switch(s) { case "completed": return "badge-success"; case "failed": return "badge-error"; case "running": return "badge-info"; default: return "badge-ghost"; }
};

function SectionHeader({ icon, title, count }: { icon: React.ReactNode; title: string; count?: number }) {
  return (
    <div className="flex items-center gap-3 mb-4">
      <div className="w-1 h-6 rounded-full bg-primary" />
      <div className="flex items-center gap-2 flex-1 min-w-0">
        <span className="text-base-content/70 shrink-0">{icon}</span>
        <h2 className="font-bold text-sm tracking-tight">{title}</h2>
      </div>
      {count !== undefined && <span className="badge badge-sm">{count}</span>}
    </div>
  );
}

export default function KanbanPage() {
  const [backends, setBackends] = useState<BackendTask[]>([]);
  const [stats, setStats] = useState<TaskStats | null>(null);
  const [localTasks, setLocalTasks] = useState<LocalTask[]>([]);
  const [newText, setNewText] = useState("");
  const [addingTo, setAddingTo] = useState<string | null>(null);

  useEffect(() => {
    const fetchTasks = () => {
      fetch("/api/talon/tasks").then(r => r.json()).then(d => { if (d.tasks) { setBackends(d.tasks); setStats(d.stats); } }).catch(() => {});
    };
    fetchTasks();
    const interval = setInterval(fetchTasks, 5000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => { try { const raw = localStorage.getItem(STORAGE_KEY); if (raw) setLocalTasks(JSON.parse(raw)); } catch {} }, []);

  const saveLocal = (next: LocalTask[]) => { setLocalTasks(next); localStorage.setItem(STORAGE_KEY, JSON.stringify(next)); };
  const addLocal = (column: "todo" | "progress" | "done") => { if (!newText.trim()) return; saveLocal([...localTasks, { id: crypto.randomUUID(), text: newText.trim(), column }]); setNewText(""); setAddingTo(null); };
  const moveTask = (id: string, to: "todo" | "progress" | "done") => saveLocal(localTasks.map(t => t.id === id ? { ...t, column: to } : t));
  const deleteTask = (id: string) => saveLocal(localTasks.filter(t => t.id !== id));

  return (
    <div className="flex flex-col gap-6">

      {/* Stats */}
      {stats && (
        <div className="grid grid-cols-5 gap-3">
          {[
            { label: "Total", value: stats.total, color: "text-base-content" },
            { label: "Running", value: stats.running, color: "text-info" },
            { label: "Completed", value: stats.completed, color: "text-success" },
            { label: "Failed", value: stats.failed, color: "text-error" },
            { label: "Pending", value: stats.pending, color: "text-base-content/70" },
          ].map(s => (
            <div key={s.label} className="rounded-xl bg-base-200 border border-base-content/10 p-3 text-center shadow-sm">
              <div className={`text-2xl font-bold ${s.color}`}>{s.value}</div>
              <div className="text-[10px] text-base-content/70 font-semibold uppercase tracking-widest mt-0.5">{s.label}</div>
            </div>
          ))}
        </div>
      )}

      {/* Agent Tasks */}
      {backends.length > 0 && (
        <div className="rounded-xl bg-base-200 border border-base-content/10 p-5 shadow-sm">
          <SectionHeader icon={<ListChecks size={16} />} title="Agent Tasks" count={backends.length} />
          <div className="flex flex-col gap-1.5 max-h-64 overflow-y-auto">
            {backends.slice().reverse().map(t => (
              <div key={t.id} className="flex items-center gap-3 px-3 py-2.5 rounded-lg bg-base-300 border border-base-content/5">
                <span className={`material-symbols-outlined text-sm ${statusColor(t.status)}`}>{statusIcon(t.status)}</span>
                <div className="flex-1 min-w-0">
                  <div className={`text-sm truncate ${t.status === "failed" ? "text-error" : "text-base-content/90"}`}>{t.description}</div>
                  {t.status === "running" && t.progress > 0 && <progress className="progress progress-info h-1.5 mt-1" value={t.progress} max="100" />}
                </div>
                <div className={`badge ${statusBadge(t.status)} badge-sm`}>{t.status}</div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Local Kanban Board */}
      <div className="rounded-xl bg-base-200 border border-base-content/10 p-5 shadow-sm">
        <SectionHeader icon={<Activity size={16} />} title="Task Board" count={localTasks.length} />
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {COLUMNS.map(col => {
            const colTasks = localTasks.filter(t => t.column === col.id);
            return (
              <div key={col.id} className="flex flex-col gap-2">
                <div className="flex items-center gap-2 mb-1">
                  <span className={`material-symbols-outlined text-sm ${col.color}`}>{col.icon}</span>
                  <span className={`text-xs font-bold uppercase tracking-wider ${col.color}`}>{col.label}</span>
                  <span className="text-xs text-base-content/60 ml-auto">{colTasks.length}</span>
                </div>
                <div className="flex flex-col gap-2 min-h-[120px] bg-base-300/50 rounded-lg p-2 border border-base-content/5">
                  {colTasks.map(task => (
                    <div key={task.id} className="group relative px-3 py-2.5 rounded-lg bg-base-300 border border-base-content/10 cursor-pointer hover:border-primary/30 transition-all"
                      onClick={() => { const next = col.id === "todo" ? "progress" : col.id === "progress" ? "done" : "todo"; moveTask(task.id, next); }}
                    >
                      <div className="text-sm font-medium text-base-content/80 pr-5">{task.text}</div>
                      <button className="absolute top-1.5 right-1.5 opacity-0 group-hover:opacity-100 btn btn-ghost btn-xs btn-square"
                        onClick={e => { e.stopPropagation(); deleteTask(task.id); }}>
                        <span className="material-symbols-outlined text-xs">close</span>
                      </button>
                    </div>
                  ))}
                  {colTasks.length === 0 && <div className="text-xs text-base-content/70 text-center py-6 italic">drop tasks here</div>}
                </div>
                {addingTo === col.id ? (
                  <div className="flex flex-col gap-2">
                    <input type="text" className="input input-bordered input-sm bg-base-300 border-base-content/20" placeholder="Task..." value={newText}
                      onChange={e => setNewText(e.target.value)}
                      onKeyDown={e => { if (e.key === "Enter") addLocal(col.id); if (e.key === "Escape") { setAddingTo(null); setNewText(""); } }} />
                    <div className="flex gap-2">
                      <button className="btn btn-primary btn-xs" onClick={() => addLocal(col.id)}>Add</button>
                      <button className="btn btn-ghost btn-xs" onClick={() => { setAddingTo(null); setNewText(""); }}>Cancel</button>
                    </div>
                  </div>
                ) : (
                  <button onClick={() => setAddingTo(col.id)} className="btn btn-ghost btn-xs w-full mt-1">
                    <span className="material-symbols-outlined text-sm">add</span> Add task
                  </button>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
