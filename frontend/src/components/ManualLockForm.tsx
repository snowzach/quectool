import { useState } from "react";

export function ManualLockForm({
  onLock,
}: {
  onLock: (tech: "4g" | "5g", freq: number, pci: number, scs?: number, band?: number) => Promise<void>;
}) {
  const [busy, setBusy] = useState(false);
  const [tech, setTech] = useState<"4g" | "5g">("5g");
  const [freq, setFreq] = useState("");
  const [pci, setPci] = useState("");
  const [scs, setScs] = useState("");
  const [band, setBand] = useState("");

  const submit = async () => {
    const f = parseInt(freq, 10);
    const p = parseInt(pci, 10);
    if (!Number.isFinite(f) || f <= 0 || !Number.isFinite(p) || p < 0) return;
    if (tech === "5g") {
      const b = parseInt(band, 10);
      if (!Number.isFinite(b) || b <= 0) return;
    }
    setBusy(true);
    try {
      const s = scs ? parseInt(scs, 10) : 0;
      const b = band ? parseInt(band, 10) : 0;
      await onLock(tech, f, p,
        Number.isFinite(s) ? s : 0,
        Number.isFinite(b) ? b : 0);
      setFreq(""); setPci(""); setScs(""); setBand("");
    } finally { setBusy(false); }
  };

  return (
    <div className="p-3 bg-slate-50 rounded space-y-2">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-2 text-sm">
        <label className="block">
          <span className="block text-xs text-slate-500 mb-0.5">Tech</span>
          <select value={tech} onChange={(e) => setTech(e.target.value as "4g" | "5g")}
            className="w-full border rounded px-2 py-1">
            <option value="4g">4G (LTE)</option>
            <option value="5g">5G (NR)</option>
          </select>
        </label>
        <label className="block">
          <span className="block text-xs text-slate-500 mb-0.5">{tech === "4g" ? "EARFCN" : "ARFCN"}</span>
          <input value={freq} onChange={(e) => setFreq(e.target.value)} inputMode="numeric"
            className="w-full border rounded px-2 py-1 font-mono" />
        </label>
        <label className="block">
          <span className="block text-xs text-slate-500 mb-0.5">PCI</span>
          <input value={pci} onChange={(e) => setPci(e.target.value)} inputMode="numeric"
            className="w-full border rounded px-2 py-1 font-mono" />
        </label>
        {tech === "5g" && (
          <>
            <label className="block">
              <span className="block text-xs text-slate-500 mb-0.5">SCS index</span>
              <input value={scs} onChange={(e) => setScs(e.target.value)} inputMode="numeric"
                placeholder="0=15kHz 1=30kHz" className="w-full border rounded px-2 py-1 font-mono" />
            </label>
            <label className="block">
              <span className="block text-xs text-slate-500 mb-0.5">Band</span>
              <input value={band} onChange={(e) => setBand(e.target.value)} inputMode="numeric"
                placeholder="e.g. 71" className="w-full border rounded px-2 py-1 font-mono" />
            </label>
          </>
        )}
      </div>
      <button onClick={() => void submit()}
        disabled={busy || !freq || !pci || (tech === "5g" && !band)}
        className="bg-accent hover:bg-accent-hover text-white rounded px-3 py-1 text-sm disabled:opacity-50">
        {busy ? "Locking…" : "Lock to entered cell"}
      </button>
    </div>
  );
}
