"use client";

import { useState, useCallback, useEffect } from "react";
import { Command } from "cmdk";
import { motion, AnimatePresence } from "framer-motion";
import { useRouter } from "next/navigation";
import {
  MessageSquare, FolderOpen, ListChecks, Brain, Bell, Puzzle, Settings, Search
} from "lucide-react";

const iconMap: Record<string, React.ReactNode> = {
  chat: <MessageSquare size={16} />,
  folder: <FolderOpen size={16} />,
  checklist: <ListChecks size={16} />,
  psychology: <Brain size={16} />,
  timeline: <Bell size={16} />,
  extension: <Puzzle size={16} />,
  settings: <Settings size={16} />,
};

const PAGES = [
  { id: "chat",      label: "Talon Chat",     icon: "chat",        href: "/chat" },
  { id: "projects",  label: "Projects",         icon: "folder",      href: "/projects" },
  { id: "kanban",    label: "Task Board",       icon: "checklist",   href: "/kanban" },
  { id: "diagrams",  label: "Diagrams",         icon: "image",       href: "/diagrams" },
  { id: "security",  label: "Security",         icon: "security",    href: "/security" },
  { id: "activity",  label: "Activity Stream",  icon: "timeline",    href: "/activity" },
  { id: "plugins",   label: "Plugins",          icon: "extension",   href: "/plugins" },
  { id: "settings",  label: "Settings",         icon: "settings",    href: "/settings" },
];

export default function CommandPalette() {
  const [open, setOpen] = useState(false);
  const router = useRouter();

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) { e.preventDefault(); setOpen(o => !o); }
    };
    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);

  const run = useCallback((href: string) => { setOpen(false); router.push(href); }, [router]);

  return (
    <AnimatePresence>
      {open && (
        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
          className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh]">
          <div className="absolute inset-0 bg-base-300/80 backdrop-blur-sm" onClick={() => setOpen(false)} />
          <motion.div initial={{ opacity: 0, scale: 0.96, y: -10 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0, scale: 0.96, y: -10 }}
            transition={{ duration: 0.15 }}
            className="relative w-full max-w-lg card bg-base-200 shadow-2xl border border-base-300 overflow-hidden">
            <Command label="Command palette" className="flex flex-col">
              <div className="flex items-center border-b border-base-300 px-4 gap-2">
                <Search size={16} className="text-base-content/60" />
                <Command.Input className="flex-1 bg-transparent outline-none py-3 text-sm" placeholder="Search pages..." autoFocus />
              </div>
              <Command.List className="p-2 max-h-64 overflow-y-auto">
                <Command.Empty className="py-8 text-center text-sm text-base-content/60">No results found.</Command.Empty>
                <Command.Group heading="Pages" className="text-[10px] uppercase tracking-widest px-2 py-1.5 text-base-content/60">
                  {PAGES.map(p => (
                    <Command.Item key={p.id} onSelect={() => run(p.href)}
                      className="flex items-center gap-3 px-3 py-2 rounded-lg text-sm cursor-pointer aria-selected:bg-primary/10 transition-colors">
                      <span className="text-primary">{iconMap[p.icon]}</span>
                      <span>{p.label}</span>
                    </Command.Item>
                  ))}
                </Command.Group>
              </Command.List>
            </Command>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
