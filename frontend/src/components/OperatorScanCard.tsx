import { useState } from "react";
import { Card } from "./Card";
import { Spinner } from "./Spinner";
import { scan } from "../api/modem";
import type { ScanResult } from "../types/modem";

export function OperatorScanCard() {
  const [busy, setBusy] = useState(false);
  const [results, setResults] = useState<ScanResult[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const start = async () => {
    setBusy(true); setError(null); setResults(null);
    try { setResults(await scan()); }
    catch (e: any) { setError(e?.message ?? "scan failed"); }
    finally { setBusy(false); }
  };

  return (
    <Card title="Operator Scan">
      <div className="text-xs text-slate-500 mb-3">
        Lists carriers visible to this SIM. Use this to confirm what networks the modem can register on.
      </div>
      <div className="flex items-center gap-3 mb-3">
        <button onClick={() => void start()} disabled={busy}
          className="bg-accent hover:bg-accent-hover text-white rounded px-3 py-1 disabled:opacity-50">
          {busy ? "Scanning…" : "Scan operators"}
        </button>
        {busy && (
          <div className="flex items-center gap-3 text-slate-600">
            <Spinner size="lg" />
            <span className="text-sm">Looking for carriers — typically 30-60 seconds.</span>
          </div>
        )}
        {error && <span className="text-red-700 text-sm">{error}</span>}
      </div>
      {results && (
        <table className="w-full text-sm">
          <thead className="text-slate-500"><tr><th className="text-left">Operator</th><th>MCC/MNC</th><th>Tech</th><th>State</th></tr></thead>
          <tbody>
            {results.map((r, i) => (
              <tr key={i} className="border-t border-slate-100">
                <td>{r.operator}</td>
                <td className="text-center">{r.mcc}/{r.mnc}</td>
                <td className="text-center">{r.tech}</td>
                <td className="text-center">{r.state}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </Card>
  );
}
