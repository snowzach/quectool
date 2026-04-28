import { ReactNode } from "react";
export function DataField({ label, value }: { label: string; value: ReactNode }) {
  const display = value === null || value === undefined || value === "" ? "—" : value;
  return (
    <div className="flex justify-between gap-4 py-1 border-b border-slate-100 last:border-b-0 text-sm">
      <span className="text-slate-500">{label}</span>
      <span className="text-slate-900 font-mono">{display}</span>
    </div>
  );
}
