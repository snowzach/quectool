// formatUptime renders seconds as "Xd Yh Zm" — used in dashboards.
export function formatUptime(s: number | undefined): string {
  if (!s || s <= 0) return "—";
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  const parts: string[] = [];
  if (d) parts.push(`${d}d`);
  if (h) parts.push(`${h}h`);
  parts.push(`${m}m`);
  return parts.join(" ");
}

// Linux Sysinfo encodes load averages as fixed-point integers scaled by 65536.
export function formatLoads(loads: number[] | undefined): string {
  if (!loads || loads.length === 0) return "—";
  return loads.map((v) => (v / 65536).toFixed(2)).join(" ");
}
