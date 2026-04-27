import { useEffect, useState, useCallback, DependencyList } from "react";

export interface ApiState<T> {
  data: T | null;
  error: Error | null;
  loading: boolean;
  reload: () => void;
}

export function useApi<T>(fn: () => Promise<T>, deps: DependencyList = []): ApiState<T> {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [loading, setLoading] = useState(true);
  const [tick, setTick] = useState(0);

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const stableFn = useCallback(fn, deps);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    stableFn().then(
      (d) => { if (!cancelled) { setData(d); setLoading(false); } },
      (e) => { if (!cancelled) { setError(e); setLoading(false); } },
    );
    return () => { cancelled = true; };
  }, [stableFn, tick]);

  return { data, error, loading, reload: () => setTick((t) => t + 1) };
}
