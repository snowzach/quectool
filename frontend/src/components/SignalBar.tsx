import { quality, qualityBarClass } from "../lib/signal";

interface Props {
  label: string;
  /** Raw value with units already stripped (number, may be negative). */
  value: number | null | undefined;
  /** Mapped 0-100 percentage from the matching `signal.ts` helper. */
  percent: number;
  /** Optional unit string appended to the raw value (e.g. "dBm"). */
  unit?: string;
}

export function SignalBar({ label, value, percent, unit }: Props) {
  const q = quality(percent);
  const bar = qualityBarClass[q];
  const display = value === null || value === undefined ? "—" : `${value}${unit ? " " + unit : ""}`;
  return (
    <div className="text-sm">
      <div className="flex justify-between mb-1">
        <span className="text-slate-600">{label}</span>
        <span className="font-mono text-slate-900">
          {display}
          {percent > 0 && <span className="text-slate-400 ml-2">{percent}%</span>}
        </span>
      </div>
      <div className="h-2 bg-slate-100 rounded overflow-hidden">
        <div
          className={`h-full transition-all duration-500 ${bar}`}
          style={{ width: `${percent}%` }}
        />
      </div>
    </div>
  );
}
