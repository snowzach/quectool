import { ToastMsg } from "../hooks/useToast";
const styles: Record<ToastMsg["kind"], string> = {
  info: "bg-slate-700 text-white",
  ok: "bg-emerald-600 text-white",
  error: "bg-red-600 text-white",
};
export function ToastStack({ toasts }: { toasts: ToastMsg[] }) {
  return (
    <div className="fixed bottom-4 right-4 flex flex-col gap-2 z-50">
      {toasts.map((t) => (
        <div key={t.id} className={`${styles[t.kind]} px-3 py-2 rounded shadow text-sm`}>{t.text}</div>
      ))}
    </div>
  );
}
