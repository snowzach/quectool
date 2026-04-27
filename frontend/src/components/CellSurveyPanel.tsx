import { useState } from "react";
import { Card } from "./Card";
import { Spinner } from "./Spinner";
import { LockConfirmModal } from "./LockConfirmModal";
import { cellSurvey } from "../api/modem";
import type { CellInfo, CellSurveyResult } from "../types/modem";

type SortKey = "rsrp" | "band" | "tech";
const TECH_ORDER: Record<string, number> = { "NR5G": 0, "LTE": 1 };

export type LockArgs = {
  tech: "4g" | "5g";
  freq: number;
  pci: number;
  scs?: number;
  band?: number;
};

export function CellSurveyPanel({
  currentCell, onLock, onError, onSuccess,
}: {
  currentCell: CellInfo | null;
  onLock: (args: LockArgs) => Promise<void>;
  onError: (msg: string) => void;
  onSuccess: (msg: string) => void;
}) {
  const [busy, setBusy] = useState(false);
  const [results, setResults] = useState<CellSurveyResult[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [sort, setSort] = useState<SortKey>("rsrp");
  const [locking, setLocking] = useState<string | null>(null);
  const [pending, setPending] = useState<CellSurveyResult | null>(null);

  const start = async () => {
    setBusy(true); setError(null); setResults(null);
    try { setResults(await cellSurvey()); }
    catch (e: any) { setError(e?.message ?? "cell survey failed"); }
    finally { setBusy(false); }
  };

  const isCurrent = (c: CellSurveyResult): boolean => {
    if (!currentCell) return false;
    if (currentCell.pci !== c.pci) return false;
    const cb = currentCell.band || "";
    const tag = c.tech === "NR5G" ? `n${c.band}` : `B${c.band}`;
    return cb === tag;
  };

  const confirmLock = async () => {
    const c = pending!;
    const tech: "4g" | "5g" = c.tech === "NR5G" ? "5g" : "4g";
    const id = `${c.freq}:${c.pci}`;
    setLocking(id);
    setPending(null);
    try {
      await onLock({
        tech,
        freq: c.freq,
        pci: c.pci,
        scs: c.tech === "NR5G" ? c.scs : undefined,
        band: c.tech === "NR5G" ? c.band : undefined,
      });
      onSuccess(`${tech.toUpperCase()} locked to ${c.tech === "NR5G" ? "n" : "B"}${c.band} PCI ${c.pci}`);
    } catch (e: any) {
      onError(e?.message ?? "lock failed");
    } finally {
      setLocking(null);
    }
  };

  const sorted = results
    ? [...results].sort((a, b) => {
        if (sort === "rsrp") return b.rsrp - a.rsrp;
        if (sort === "band") return a.band - b.band || b.rsrp - a.rsrp;
        return (TECH_ORDER[a.tech] ?? 9) - (TECH_ORDER[b.tech] ?? 9) || b.rsrp - a.rsrp;
      })
    : null;

  return (
    <Card title="Cell Survey">
      <div className="text-xs text-slate-500 mb-3">
        Active RF sweep — every cell the modem can hear, with per-cell RSRP/RSRQ.
        Slower than the operator scan (~60-120 seconds) and disrupts the
        current connection while running.
      </div>
      <div className="flex items-center gap-3 mb-3">
        <button onClick={() => void start()} disabled={busy}
          className="bg-accent hover:bg-accent-hover text-white rounded px-3 py-1 disabled:opacity-50">
          {busy ? "Surveying…" : "Run survey"}
        </button>
        {busy && (
          <div className="flex items-center gap-3 text-slate-600">
            <Spinner size="lg" />
            <span className="text-sm">Sweeping all bands — this can take up to 2 minutes.</span>
          </div>
        )}
        {error && <span className="text-red-700 text-sm">{error}</span>}
      </div>
      {sorted && sorted.length === 0 && (
        <div className="text-sm text-slate-500">No cells reported.</div>
      )}
      {sorted && sorted.length > 0 && (
        <>
          <div className="flex items-center gap-2 mb-2 text-xs">
            <span className="text-slate-500">Sort by:</span>
            {(["rsrp", "band", "tech"] as SortKey[]).map((k) => (
              <button key={k} type="button" onClick={() => setSort(k)}
                className={`rounded px-2 py-0.5 ${sort === k
                  ? "bg-accent text-white"
                  : "bg-slate-100 text-slate-600 hover:bg-slate-200"}`}>
                {k === "rsrp" ? "Signal" : k.charAt(0).toUpperCase() + k.slice(1)}
              </button>
            ))}
            <span className="ml-auto text-slate-500">{sorted.length} cells</span>
          </div>
          <table className="w-full text-sm">
            <thead className="text-slate-500 text-xs">
              <tr>
                <th className="text-left">Tech</th>
                <th className="text-left">Band</th>
                <th className="text-left">PCI</th>
                <th className="text-right">RSRP</th>
                <th className="text-right">RSRQ</th>
                <th className="text-left">MCC/MNC</th>
                <th className="text-right">Freq</th>
                <th className="text-left">Cell ID</th>
                <th className="text-right"></th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((c, i) => {
                const here = isCurrent(c);
                return (
                  <tr key={i} className={`border-t border-slate-100 ${here ? "bg-amber-50" : ""}`}>
                    <td>
                      <span className={`inline-block px-1.5 py-0.5 rounded text-xs font-semibold border ${
                        c.tech === "NR5G"
                          ? "bg-violet-100 text-violet-800 border-violet-200"
                          : "bg-sky-100 text-sky-800 border-sky-200"
                      }`}>{c.tech}</span>
                      {here && <span className="ml-2 text-xs text-amber-700">(current)</span>}
                    </td>
                    <td className="font-mono">{c.tech === "NR5G" ? "n" : "B"}{c.band || "?"}</td>
                    <td className="font-mono">{c.pci}</td>
                    <td className={`text-right font-mono ${
                      c.rsrp >= -90 ? "text-emerald-700"
                        : c.rsrp >= -105 ? "text-amber-700"
                        : "text-red-700"
                    }`}>{c.rsrp || "—"}</td>
                    <td className="text-right font-mono">{c.rsrq || "—"}</td>
                    <td className="font-mono text-xs">{c.mcc}/{c.mnc}</td>
                    <td className="text-right font-mono text-xs">{c.freq}</td>
                    <td className="font-mono text-xs">{c.cell_id}</td>
                    <td className="text-right">
                      <button onClick={() => setPending(c)}
                        disabled={locking === `${c.freq}:${c.pci}`}
                        className="text-xs bg-slate-100 hover:bg-accent hover:text-white text-slate-700 rounded px-2 py-0.5 disabled:opacity-40">
                        {locking === `${c.freq}:${c.pci}` ? "Locking…" : "Lock"}
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </>
      )}
      {pending && (
        <LockConfirmModal
          title={`Lock ${pending.tech === "NR5G" ? "5G" : "4G"} to this cell?`}
          body={
            <>
              <div className="font-mono">
                {pending.tech} {pending.tech === "NR5G" ? "n" : "B"}{pending.band} ·
                PCI {pending.pci} · RSRP {pending.rsrp} dBm
              </div>
              <div className="mt-2 text-slate-600">
                Pins the modem to this exact cell. If you move out of range
                you'll lose service until you unlock.
              </div>
            </>
          }
          confirmLabel="Lock"
          onConfirm={() => void confirmLock()}
          onCancel={() => setPending(null)}
        />
      )}
    </Card>
  );
}
