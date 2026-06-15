"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState, useEffect, useRef } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { LayoutDashboard, ListChecks, Wrench, Moon, Sun, Radar, Image, Network, Shield } from "lucide-react";

interface NavItem {
  href: string;
  label: string;
  icon: typeof LayoutDashboard;
}

const NAV_ITEMS: NavItem[] = [
  { href: "/",       label: "Dashboard", icon: LayoutDashboard },
  { href: "/kanban", label: "Tasks",     icon: ListChecks },
  { href: "/memory", label: "Code Graph", icon: Network },
  { href: "/diagrams", label: "Diagrams", icon: Image },
  { href: "/tools",  label: "Tools",     icon: Wrench },
  { href: "/security", label: "Security", icon: Shield },
];

const itemVariants = {
  hidden: { opacity: 0, x: -12, scale: 0.9 },
  visible: (i: number) => ({
    opacity: 1, x: 0, scale: 1,
    transition: { type: "spring" as const, stiffness: 400, damping: 28, delay: i * 0.04 },
  }),
};

export default function Navigation() {
  const pathname = usePathname();
  const [expanded, setExpanded] = useState(false);
  const [theme, setTheme] = useState<"halloween" | "emerald">("halloween");
  const [mounted, setMounted] = useState(false);
  const barRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setMounted(true);
    const saved = localStorage.getItem("talon-theme") as "halloween" | "emerald" | null;
    const t = saved || "halloween";
    setTheme(t);
    document.documentElement.setAttribute("data-theme", t);
  }, []);

  // Close on click outside
  useEffect(() => {
    if (!expanded) return;
    const handler = (e: MouseEvent) => {
      if (barRef.current && !barRef.current.contains(e.target as Node)) {
        setExpanded(false);
      }
    };
    // Small delay so the click that opened it doesn't immediately close it
    const timer = setTimeout(() => document.addEventListener("click", handler), 50);
    return () => { clearTimeout(timer); document.removeEventListener("click", handler); };
  }, [expanded]);

  const toggleTheme = () => {
    const next = theme === "halloween" ? "emerald" : "halloween";
    setTheme(next);
    localStorage.setItem("talon-theme", next);
    document.documentElement.setAttribute("data-theme", next);
  };

  const isActive = (href: string) => {
    if (href === "/") return pathname === "/";
    return pathname.startsWith(href);
  };

  const navItem = (item: NavItem, i: number) => {
    const active = isActive(item.href);
    const Icon = item.icon;
    return (
      <motion.div
        key={item.href}
        custom={i}
        variants={itemVariants}
        initial="hidden"
        animate="visible"
      >
        <Link
          href={item.href}
          onClick={() => setExpanded(false)}
          className={`flex items-center gap-2 px-3 py-2 rounded-xl text-sm font-medium transition-all whitespace-nowrap ${
            active
              ? "bg-primary text-primary-content shadow-sm"
              : "text-base-content/70 hover:text-base-content hover:bg-base-300/50"
          }`}
        >
          <Icon size={18} />
          <span>{item.label}</span>
        </Link>
      </motion.div>
    );
  };

  if (!mounted) return null;

  return (
    <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50" ref={barRef}>
      <motion.div
        layout
        className="flex items-center gap-1 rounded-2xl border border-base-content/10 shadow-lg backdrop-blur-xl px-1.5 py-1.5"
        style={{ background: "color-mix(in oklab, var(--color-base-200) 85%, transparent)" }}
      >
        {/* Toggle button — always visible */}
        <motion.button
          layout
          onClick={() => setExpanded(!expanded)}
          className="flex items-center gap-2 px-3 py-2 rounded-xl hover:bg-base-300/50 transition-colors"
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.97 }}
        >
          <Radar size={20} className="text-primary" />
          <AnimatePresence mode="wait">
            {!expanded && (
              <motion.span
                initial={{ opacity: 0, width: 0 }}
                animate={{ opacity: 1, width: "auto" }}
                exit={{ opacity: 0, width: 0 }}
                className="text-sm font-bold overflow-hidden whitespace-nowrap"
              >
                Talon
              </motion.span>
            )}
          </AnimatePresence>
        </motion.button>

        {/* Expanded nav items */}
        <AnimatePresence>
          {expanded && (
            <>
              <div className="w-px h-6 bg-base-content/10 mx-1" />
              {NAV_ITEMS.map((item, i) => navItem(item, i))}
              <div className="w-px h-6 bg-base-content/10 mx-1" />
              {/* Theme toggle */}
              <motion.div
                custom={NAV_ITEMS.length}
                variants={itemVariants}
                initial="hidden"
                animate="visible"
              >
                <button
                  onClick={toggleTheme}
                  className="flex items-center gap-2 px-3 py-2 rounded-xl text-sm font-medium text-base-content/70 hover:text-base-content hover:bg-base-300/50 transition-all whitespace-nowrap"
                  title={theme === "halloween" ? "Switch to light" : "Switch to dark"}
                >
                  {theme === "halloween" ? <Sun size={18} /> : <Moon size={18} />}
                  <span className="hidden sm:inline">{theme === "halloween" ? "Light" : "Dark"}</span>
                </button>
              </motion.div>
            </>
          )}
        </AnimatePresence>

        {/* Chevron indicator */}
        {!expanded && (
          <span className="material-symbols-outlined text-base-content/30 text-sm">keyboard_arrow_up</span>
        )}
      </motion.div>
    </div>
  );
}
