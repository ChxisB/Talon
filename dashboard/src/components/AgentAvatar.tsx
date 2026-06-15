"use client";

import { motion } from "framer-motion";

export type AgentKey = "talon" | "claude" | "fcc";

interface Props {
  agent: AgentKey;
  size?: number;
  pulse?: boolean;
}

const STYLE: Record<AgentKey, {
  accent: string;
  bg: string;
  gradient: string;
  label: string;
  glyph: (size: number) => React.ReactNode;
}> = {
  talon: {
    accent: "#14b8a6",
    bg: "rgba(20,184,166,0.18)",
    gradient: "linear-gradient(135deg, #2dd4bf 0%, #14b8a6 60%, #0d9488 100%)",
    label: "Talon",
    glyph: (s) => (
      <svg width={s * 0.55} height={s * 0.55} viewBox="0 0 24 24" fill="none">
        <path
          d="M12 2 L15 9 L22 9 L16 14 L18 21 L12 16 L6 21 L8 14 L2 9 L9 9 Z"
          fill="white"
          opacity="0.95"
        />
      </svg>
    ),
  },
  claude: {
    accent: "#d97757",
    bg: "rgba(217,119,87,0.18)",
    gradient: "linear-gradient(135deg, #f4a07a, #c0563a)",
    label: "Claude",
    glyph: (s) => (
      <svg width={s * 0.55} height={s * 0.55} viewBox="0 0 24 24" fill="none">
        <path
          d="M12 2 L13.6 9 L21 10.4 L13.6 12.4 L12 22 L10.4 12.4 L3 10.4 L10.4 9 Z"
          fill="white"
          opacity="0.95"
        />
      </svg>
    ),
  },
  fcc: {
    accent: "#10b981",
    bg: "rgba(16,185,129,0.18)",
    gradient: "linear-gradient(135deg, #34d399 0%, #10b981 60%, #065f46 100%)",
    label: "Free Claude Code",
    glyph: (s) => (
      <svg width={s * 0.62} height={s * 0.62} viewBox="0 0 24 24" fill="none">
        <path d="M5 4 L7 7 M19 4 L17 7" stroke="white" strokeWidth="1.6" strokeLinecap="round" opacity="0.95" />
        <circle cx="9" cy="11" r="3.2" fill="white" opacity="0.95" />
        <circle cx="15" cy="11" r="3.2" fill="white" opacity="0.95" />
        <circle cx="9" cy="11" r="1.2" fill="#065f46" />
        <circle cx="15" cy="11" r="1.2" fill="#065f46" />
        <path d="M11 14 L12 16 L13 14 Z" fill="white" opacity="0.9" />
        <path d="M6 17 C 8 19, 16 19, 18 17" stroke="white" strokeWidth="1.4" strokeLinecap="round" fill="none" opacity="0.85" />
      </svg>
    ),
  },
};

export default function AgentAvatar({ agent, size = 36, pulse = false }: Props) {
  const s = STYLE[agent] ?? STYLE.talon;
  return (
    <motion.span
      initial={{ scale: 0.85, opacity: 0 }}
      animate={{ scale: 1, opacity: 1 }}
      whileHover={{ scale: 1.05 }}
      transition={{ type: "spring", stiffness: 380, damping: 25 }}
      className="relative inline-grid place-items-center rounded-full overflow-hidden shrink-0"
      style={{
        width: size,
        height: size,
        background: s.gradient,
        boxShadow: `0 0 ${size}px -${size / 3}px ${s.accent}, inset 0 0 0 1px rgba(255,255,255,0.12)`,
      }}
      aria-label={s.label}
    >
      {s.glyph(size)}
      {pulse && (
        <span
          className="absolute inset-0 rounded-full pointer-events-none"
          style={{
            boxShadow: `0 0 0 0 ${s.accent}`,
            animation: "avatar-pulse 1.8s ease-out infinite",
          }}
        />
      )}
      <style jsx>{`
        @keyframes avatar-pulse {
          0%   { box-shadow: 0 0 0 0 ${s.accent}88; }
          70%  { box-shadow: 0 0 0 ${size * 0.5}px transparent; }
          100% { box-shadow: 0 0 0 0 transparent; }
        }
      `}</style>
    </motion.span>
  );
}

export function agentColor(agent: AgentKey): string {
  return STYLE[agent]?.accent ?? STYLE.talon.accent;
}
export function agentLabel(agent: AgentKey): string {
  return STYLE[agent]?.label ?? STYLE.talon.label;
}
