import { useState } from "react";
import { Card } from "./Card";
import type { CellInfo, CellLockSettings } from "../types/modem";

type Tech = "4g" | "5g";

export function NetworkStatusStrip({
  cell, cellLock, onLockCurrent, onUnlock,
}: {
  cell: CellInfo | null;
  cellLock: CellLockSettings;
  onLockCurrent: (tech: Tech) => Promise<void>;
  onUnlock: (tech: Tech) => Promise<void>;
}) {
  const [busy, setBusy] = useState<string | null>(null);

  const click = async (tech: Tech, action: "lock" | "unlock") => {
    setBusy(`${tech}-${action}`);
    try {
      if (action === "lock") await onLockCurrent(tech);
      else await onUnlock(tech);
    } finally {
      setBusy(null);
    }
  };

  if (!cell) {
    return (
      <Card>
        <div className="text-amber-700 text-sm">Acquiring serving-cell info…</div>
      </Card>
    );
  }

  const sig = cell.signal;
  const scells = cell.scells ?? [];

  const Chip = ({ tech }: { tech: Tech }) => {
    const locked = tech === "4g" ? cellLock.lock_4g : cellLock.lock_5g;
    const busyKey = `${tech}-${locked ? "unlock" : "lock"}`;
    return (
      <div
        className={`inline-flex items-center gap-2 rounded border px-2 py-1 text-xs ${
          locked ? "border-emerald-200 bg-emerald-50" : "border-slate-200 bg-slate-50"
        }`}
      >
        <span className={`font-semibold ${locked ? "text-emerald-700" : "text-slate-600"}`}>
          {tech.toUpperCase()} {locked ? "LOCKED" : "FREE"}
        </span>
        {locked ? (
          <button
            onClick={() => void click(tech, "unlock")}
            disabled={busy === busyKey}
            className="bg-slate-100 hover:bg-slate-200 text-slate-700 rounded px-2 py-0.5 disabled:opacity-50"
            aria-label={`Unlock ${tech.toUpperCase()}`}
          >
            {busy === busyKey ? "Unlocking…" : "Unlock"}
          </button>
        ) : (
          <button
            onClick={() => void click(tech, "lock")}
            disabled={busy === busyKey}
            className="bg-accent hover:bg-accent-hover text-white rounded px-2 py-0.5 disabled:opacity-50"
            aria-label={`Lock ${tech.toUpperCase()} to current cell`}
          >
            {busy === busyKey ? "Locking…" : "Lock to current"}
          </button>
        )}
      </div>
    );
  };

  return (
    <Card>
      <div className="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm">
        <div className="flex items-baseline gap-2">
          <span className="text-xs uppercase tracking-wide text-slate-500">On</span>
          <span className="font-mono font-semibold">{cell.tech}</span>
          <span className="font-mono">{cell.band || "?"}</span>
          <span className="font-mono text-slate-500">PCI {cell.pci || "—"}</span>
        </div>
        <div className="flex items-baseline gap-2 font-mono">
          <span>{sig?.rsrp ?? "—"}</span>
          <span className="text-xs text-slate-500">dBm</span>
          {sig?.sinr !== undefined && <span className="text-slate-500">SINR {sig.sinr}</span>}
        </div>
        {scells.length > 0 && (
          <div className="flex flex-wrap items-center gap-1">
            <span className="text-xs text-slate-500">CA</span>
            {scells.map((s, i) => (
              <span key={i} className="font-mono text-xs bg-slate-100 rounded px-1.5 py-0.5">
                + {s.band || "?"}
              </span>
            ))}
          </div>
        )}
        <div className="flex flex-wrap items-center gap-2 ml-auto">
          <Chip tech="5g" />
          <Chip tech="4g" />
        </div>
      </div>
    </Card>
  );
}
