"use client";

import { usePathname } from "next/navigation";
import { useState, useEffect } from "react";

const TITLES: Record<string, { title: string; sub: string }> = {
  "/":         { title: "Dashboard", sub: "Agent status, task stats, and quick navigation." },
  "/kanban":   { title: "Tasks",     sub: "Track agent tasks and manage your workflow." },
  "/memory":   { title: "Code Graph", sub: "Analyse your codebase — functions, types, files and their relationships." },
  "/diagrams": { title: "Diagrams",  sub: "Architecture, workflow, sequence, and dataflow diagrams." },
  "/tools":    { title: "Tools",     sub: "API keys, plugins, cron jobs, MCP servers, and configuration." },
  "/activity": { title: "Activity",  sub: "Full event log of agent actions and system events." },
  "/security": { title: "Security",  sub: "CVE vulnerability database with search, filters, and severity tracking." },
};

export default function TopAppBar() {
  const pathname = usePathname();
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 12);
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  const t = TITLES[pathname] ?? TITLES["/"];

  return (
    <header
      className="w-full transition-all duration-200 rounded-box px-4 py-4 -mx-4"
      style={{
        background: scrolled ? "var(--color-base-200)" : "transparent",
        borderBottom: scrolled ? "1px solid color-mix(in oklab, var(--color-base-content) 15%, transparent)" : "1px solid transparent",
      }}
    >
      <h1 className="text-3xl sm:text-4xl font-bold tracking-tight">{t.title}</h1>
      <p className="text-sm text-base-content/60 mt-1 max-w-xl hidden sm:block">{t.sub}</p>
    </header>
  );
}
