import { useEffect, useRef } from "react";

/**
 * Polls `fn` every `intervalMs` while the tab is visible.
 * Pauses immediately when the tab becomes hidden,
 * fires once on return, and never spawns overlapping calls.
 */
export function usePollWhileVisible(fn: () => void, intervalMs: number) {
  const fnRef = useRef(fn);
  fnRef.current = fn;

  useEffect(() => {
    let active = true;
    let id: ReturnType<typeof setTimeout> | undefined;

    const tick = () => {
      if (!active) return;
      fnRef.current();
      id = setTimeout(tick, intervalMs);
    };

    const onVisibility = () => {
      if (document.hidden) {
        if (id !== undefined) clearTimeout(id);
      } else {
        tick(); // immediate + schedule next
      }
    };

    tick();
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      active = false;
      if (id !== undefined) clearTimeout(id);
      document.removeEventListener("visibilitychange", onVisibility);
    };
  }, [intervalMs]);
}
