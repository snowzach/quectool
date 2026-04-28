import { useCallback, useState } from "react";

export interface ToastMsg { id: number; kind: "info" | "error" | "ok"; text: string }

let nextId = 1;

export function useToast() {
  const [toasts, setToasts] = useState<ToastMsg[]>([]);
  const push = useCallback((kind: ToastMsg["kind"], text: string) => {
    const id = nextId++;
    setToasts((t) => [...t, { id, kind, text }]);
    setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), 4000);
  }, []);
  return { toasts, info: (t: string) => push("info", t), error: (t: string) => push("error", t), ok: (t: string) => push("ok", t) };
}
