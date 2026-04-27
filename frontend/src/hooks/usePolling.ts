import { useEffect, useState, useCallback, DependencyList } from "react";

export function usePolling<T>(fn: () => Promise<T>, intervalMs: number, deps: DependencyList = []) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const stable = useCallback(fn, deps);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | null = null;
    const run = async () => {
      try {
        const d = await stable();
        if (!cancelled) { setData(d); setError(null); }
      } catch (e) {
        if (!cancelled) setError(e as Error);
      } finally {
        if (!cancelled) timer = setTimeout(run, intervalMs);
      }
    };
    void run();
    return () => { cancelled = true; if (timer) clearTimeout(timer); };
  }, [stable, intervalMs]);

  return { data, error };
}
