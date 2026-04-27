import { useState, FormEvent } from "react";
import { Card } from "../components/Card";
import { Spinner } from "../components/Spinner";
import { atcmd } from "../api/atcmd";
import type { ATResponse } from "../api/atcmd";
import { useDocumentTitle } from "../hooks/useDocumentTitle";

type Preset = { label: string; cmd: string; help?: string };
type PresetGroup = { group: string; items: Preset[] };

const PRESETS: PresetGroup[] = [
  {
    group: "Identification",
    items: [
      { label: "AT", cmd: "AT", help: "Basic ping" },
      { label: "ATI", cmd: "ATI", help: "Manufacturer / model / firmware" },
      { label: "Manufacturer", cmd: "AT+CGMI" },
      { label: "Model", cmd: "AT+CGMM" },
      { label: "Firmware", cmd: "AT+CGMR" },
      { label: "IMEI", cmd: "AT+CGSN" },
    ],
  },
  {
    group: "SIM",
    items: [
      { label: "SIM ready?", cmd: "AT+CPIN?" },
      { label: "ICCID", cmd: "AT+QCCID" },
      { label: "IMSI", cmd: "AT+CIMI" },
      { label: "Subscriber #", cmd: "AT+CNUM" },
    ],
  },
  {
    group: "Network",
    items: [
      { label: "Operator", cmd: "AT+COPS?" },
      { label: "LTE registration", cmd: "AT+CEREG?" },
      { label: "5G registration", cmd: "AT+C5GREG?" },
      { label: "PS attach", cmd: "AT+CGATT?" },
      { label: "PDP contexts", cmd: "AT+CGDCONT?" },
      { label: "PDP state", cmd: "AT+CGACT?" },
      { label: "Signal (CSQ)", cmd: "AT+CSQ" },
    ],
  },
  {
    group: "Cell info",
    items: [
      { label: "Serving cell", cmd: 'AT+QENG="servingcell"' },
      { label: "Neighbours", cmd: 'AT+QENG="neighbourcell"' },
      { label: "Carrier agg", cmd: "AT+QCAINFO" },
    ],
  },
  {
    group: "Config",
    items: [
      { label: "Mode pref", cmd: 'AT+QNWPREFCFG="mode_pref"' },
      { label: "5G disable mode", cmd: 'AT+QNWPREFCFG="nr5g_disable_mode"' },
      { label: "LTE bands", cmd: 'AT+QNWPREFCFG="lte_band"' },
      { label: "5G bands", cmd: 'AT+QNWPREFCFG="nr5g_band"' },
      { label: "4G lock", cmd: 'AT+QNWLOCK="common/4g"' },
      { label: "5G lock", cmd: 'AT+QNWLOCK="common/5g"' },
      { label: "USB net", cmd: 'AT+QCFG="usbnet"' },
      { label: "Data format", cmd: 'AT+QCFG="data_interface"' },
    ],
  },
  {
    group: "Diagnostics",
    items: [
      { label: "CFUN?", cmd: "AT+CFUN?", help: "Radio state" },
      { label: "Temperature", cmd: "AT+QTEMP" },
    ],
  },
];

export function ATCmd() {
  useDocumentTitle("AT");
  const [cmd, setCmd] = useState("AT");
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<ATResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const run = async (toRun: string) => {
    setBusy(true); setError(null);
    try { setResult(await atcmd(toRun)); }
    catch (e: any) { setError(e?.message ?? "failed"); }
    finally { setBusy(false); }
  };

  const onSubmit = (e: FormEvent) => { e.preventDefault(); void run(cmd); };

  return (
    <div className="space-y-4">
      <Card title="AT Command">
        <form onSubmit={onSubmit} className="flex gap-2 mb-3">
          <input className="flex-1 border rounded px-2 py-1 font-mono" value={cmd} onChange={(e) => setCmd(e.target.value)} />
          <button disabled={busy} className="bg-accent hover:bg-accent-hover text-white rounded px-3 py-1 disabled:opacity-50">
            {busy ? <Spinner /> : "Send"}
          </button>
        </form>
        {error && <div className="text-red-700 text-sm mb-2">{error}</div>}
        {result && (
          <pre className="bg-slate-900 text-slate-100 rounded p-3 text-xs whitespace-pre-wrap font-mono">
{[result.command, ...(result.response ?? []), String(result.status)].join("\n")}
          </pre>
        )}
      </Card>

      <Card title="Presets">
        <div className="text-xs text-slate-500 mb-3">
          Click to run. Loads the command into the box above and shows the response.
        </div>
        <div className="space-y-3">
          {PRESETS.map((g) => (
            <div key={g.group}>
              <div className="text-xs uppercase tracking-wide text-slate-500 mb-1">{g.group}</div>
              <div className="flex flex-wrap gap-1">
                {g.items.map((p) => (
                  <button key={p.cmd} type="button" disabled={busy}
                    title={p.help ? `${p.cmd} — ${p.help}` : p.cmd}
                    onClick={() => { setCmd(p.cmd); void run(p.cmd); }}
                    className="text-xs font-mono bg-slate-100 hover:bg-slate-200 border border-slate-200 text-slate-700 rounded px-2 py-1 disabled:opacity-50">
                    {p.label}
                  </button>
                ))}
              </div>
            </div>
          ))}
        </div>
      </Card>
    </div>
  );
}
