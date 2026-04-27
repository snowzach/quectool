// Signal-quality math ported from the legacy dashboard.
//
// Each metric has an empirical "useful range" mapped to 0-100%:
//   RSRP : -135 dBm (poor)  →  -65 dBm (excellent)
//   RSRQ :  -20 dB           →   -8 dB
//   SINR :  -10 dB           →   35 dB
//
// Below the floor returns 0 ("no signal"); inside the range we clamp
// the bottom to 15% so a bar with any signal is at least visible.

const clamp = (n: number, min: number, max: number) =>
  Math.max(min, Math.min(max, n));

function pct(value: number, min: number, max: number, floor: number) {
  if (!Number.isFinite(value) || value < floor) return 0;
  const raw = ((value - min) / (max - min)) * 100;
  return Math.round(clamp(raw, 15, 100));
}

export const rsrpPercent = (rsrp: number) => pct(rsrp, -135, -65, -140);
export const rsrqPercent = (rsrq: number) => pct(rsrq, -20, -8, -20);
export const sinrPercent = (sinr: number) => pct(sinr, -10, 35, -10);

/** Combined quality estimate: average of RSRP and SINR percentages. */
export function overallPercent(rsrp: number, sinr: number): number {
  const r = rsrpPercent(rsrp);
  const s = sinrPercent(sinr);
  if (r === 0 && s === 0) return 0;
  return Math.round((r + s) / 2);
}

export type Quality = "excellent" | "good" | "fair" | "poor" | "none";

export function quality(percent: number): Quality {
  if (percent >= 80) return "excellent";
  if (percent >= 60) return "good";
  if (percent >= 40) return "fair";
  if (percent > 0) return "poor";
  return "none";
}

export const qualityLabel: Record<Quality, string> = {
  excellent: "Excellent",
  good: "Good",
  fair: "Fair",
  poor: "Poor",
  none: "No Signal",
};

/** Tailwind classes per quality band — bar fill colors. */
export const qualityBarClass: Record<Quality, string> = {
  excellent: "bg-emerald-500",
  good: "bg-emerald-500",
  fair: "bg-amber-500",
  poor: "bg-red-500",
  none: "bg-slate-300",
};

/** Tailwind classes per quality band — text accents. */
export const qualityTextClass: Record<Quality, string> = {
  excellent: "text-emerald-700",
  good: "text-emerald-700",
  fair: "text-amber-700",
  poor: "text-red-700",
  none: "text-slate-500",
};
