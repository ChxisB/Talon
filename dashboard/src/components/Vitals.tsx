"use client";

import { motion } from "framer-motion";
import AgentAvatar from "./AgentAvatar";

interface VitalTileProps {
  icon: React.ReactNode;
  label: string;
  value: string;
  sub?: string;
  status?: "ok" | "warn" | "err";
}

function VitalTile({ icon, label, value, sub, status = "ok" }: VitalTileProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      className="vital-tile"
    >
      <div className="flex items-center justify-between">
        <span className="k">{label}</span>
        <span className={`status-dot ${status}`} />
      </div>
      <div className="v flex items-center gap-2">
        <span className="shrink-0" style={{ color: status === "ok" ? "var(--emerald)" : "var(--gold)" }}>
          {icon}
        </span>
        <span>{value}</span>
      </div>
      {sub && <span className="sub">{sub}</span>}
    </motion.div>
  );
}

export default function Vitals() {
  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
      <VitalTile
        icon={<AgentAvatar agent="talon" size={20} />}
        label="Talon"
        value="Online"
        sub="v0.1 · Proxy 8082"
        status="ok"
      />
      <VitalTile
        icon={<AgentAvatar agent="claude" size={20} />}
        label="Claude Code"
        value="Connected"
        sub="via Free Claude Code"
        status="ok"
      />
      <VitalTile
        icon={<span className="text-[10px] mono">LLM</span>}
        label="Provider"
        value="OpenRouter"
        sub="200+ models available"
        status="ok"
      />
      <VitalTile
        icon={<span className="text-[10px] mono">DB</span>}
        label="Session Store"
        value="SQLite"
        sub="Local persistence"
        status="ok"
      />
    </div>
  );
}
