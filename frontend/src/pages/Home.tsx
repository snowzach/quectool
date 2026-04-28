import { Card } from "../components/Card";
import { DataField } from "../components/DataField";
import { Spinner } from "../components/Spinner";
import { SignalBar } from "../components/SignalBar";
import { useApi } from "../hooks/useApi";
import { usePolling } from "../hooks/usePolling";
import { getDashboard } from "../api/dashboard";
import { getSignal, getInfo } from "../api/modem";
import {
  rsrpPercent, rsrqPercent, sinrPercent, overallPercent,
  quality, qualityLabel, qualityTextClass, qualityBarClass,
} from "../lib/signal";
import { formatUptime, formatLoads } from "../lib/format";
import { useDocumentTitle } from "../hooks/useDocumentTitle";

function techBadge(tech: string | undefined) {
  if (!tech) return null;
  const cls = tech.startsWith("NR5G")
    ? "bg-violet-100 text-violet-800 border-violet-200"
    : tech === "LTE"
      ? "bg-sky-100 text-sky-800 border-sky-200"
      : "bg-slate-100 text-slate-700 border-slate-200";
  return (
    <span className={`inline-block px-2 py-0.5 rounded text-xs font-semibold border ${cls}`}>
      {tech}
    </span>
  );
}

export function Home() {
  useDocumentTitle("");
  const { data: dash, error, loading } = useApi(getDashboard);
  const { data: info } = useApi(getInfo);
  const { data: live } = usePolling(getSignal, 5_000);

  if (loading) return <div className="flex items-center gap-2 text-slate-500"><Spinner /> Loading…</div>;
  if (error) return <div className="text-red-700">{error.message}</div>;
  if (!dash) return null;

  const sig = live ?? dash.signal;
  const errs = dash.errors ?? {};

  const rsrp = sig?.rsrp ?? NaN;
  const rsrq = sig?.rsrq ?? NaN;
  const sinr = sig?.sinr ?? NaN;
  const overall = sig ? overallPercent(rsrp, sinr) : 0;
  const q = quality(overall);

  // Modem QENG state: SEARCH / NOSRV / LIMSRV mean we're not on a cell yet —
  // render "Acquiring…" instead of a misleading "0% / Offline".
  const acquiring = sig?.state === "SEARCH" || sig?.state === "NOSRV" || sig?.state === "LIMSRV";
  const registered = dash.sim?.registered === true;

  return (
    <div className="space-y-4">
      {/* Hero: signal % + connection status */}
      <div className="grid md:grid-cols-2 gap-4">
        <Card>
          <div className="flex items-center justify-between">
            <div>
              <div className="text-xs uppercase tracking-wide text-slate-500 mb-1">Signal</div>
              {acquiring ? (
                <>
                  <div className="text-3xl font-bold text-amber-700">Acquiring…</div>
                  <div className="text-sm text-slate-500 mt-1">Modem is searching for a cell ({sig?.state})</div>
                </>
              ) : (
                <>
                  <div className="flex items-baseline gap-3">
                    <span className={`text-5xl font-bold ${qualityTextClass[q]}`}>{overall}%</span>
                    <span className={`text-sm font-medium ${qualityTextClass[q]}`}>{qualityLabel[q]}</span>
                  </div>
                  <div className="mt-2">{techBadge(sig?.tech)}</div>
                </>
              )}
            </div>
            <div className={`w-3 h-20 rounded ${acquiring ? "bg-amber-400" : qualityBarClass[q]}`} />
          </div>
        </Card>
        <Card>
          <div className="text-xs uppercase tracking-wide text-slate-500 mb-1">Connection</div>
          {errs.sim ? (
            <div className="text-red-700 text-sm">{errs.sim}</div>
          ) : (
            <>
              <div className="flex items-baseline gap-3">
                <span className={`text-3xl font-bold ${
                  registered ? "text-emerald-700" : acquiring ? "text-amber-700" : "text-red-700"
                }`}>
                  {registered ? "Connected" : acquiring ? "Searching" : "Offline"}
                </span>
              </div>
              <div className="mt-3 grid grid-cols-2 gap-x-4 text-sm">
                <div className="text-slate-500">Operator</div>
                <div className="font-mono text-slate-900 text-right">{dash.sim?.operator || "—"}</div>
                <div className="text-slate-500">APN</div>
                <div className="font-mono text-slate-900 text-right">{dash.sim?.apn || "—"}</div>
                {dash.sim?.ipv4 && (
                  <>
                    <div className="text-slate-500">IPv4</div>
                    <div className="font-mono text-slate-900 text-right">{dash.sim.ipv4}</div>
                  </>
                )}
                {dash.sim?.ipv6 && (
                  <>
                    <div className="text-slate-500">IPv6</div>
                    <div className="font-mono text-slate-900 text-right text-xs break-all">{dash.sim.ipv6}</div>
                  </>
                )}
                {dash.sim?.gateway && (
                  <>
                    <div className="text-slate-500">Gateway</div>
                    <div className="font-mono text-slate-900 text-right">{dash.sim.gateway}</div>
                  </>
                )}
                {dash.sim?.dns && dash.sim.dns.length > 0 && (
                  <>
                    <div className="text-slate-500">DNS</div>
                    <div className="font-mono text-slate-900 text-right text-xs">
                      {dash.sim.dns.map((d) => <div key={d} className="break-all">{d}</div>)}
                    </div>
                  </>
                )}
              </div>
            </>
          )}
        </Card>
      </div>

      {/* Signal bars */}
      <Card title="Signal Strength">
        {errs.signal ? <div className="text-red-700 text-sm">{errs.signal}</div> : (
          <div className="space-y-3">
            <SignalBar label="RSRP" value={Number.isFinite(rsrp) ? rsrp : null} percent={rsrpPercent(rsrp)} unit="dBm" />
            <SignalBar label="RSRQ" value={Number.isFinite(rsrq) ? rsrq : null} percent={rsrqPercent(rsrq)} unit="dB" />
            <SignalBar label="SINR" value={Number.isFinite(sinr) ? sinr : null} percent={sinrPercent(sinr)} unit="dB" />
          </div>
        )}
      </Card>

      {/* Detail cards */}
      <div className="grid md:grid-cols-2 gap-4">
        <Card title="Cell">
          {errs.cell ? <div className="text-red-700 text-sm">{errs.cell}</div> : (
            <>
              <DataField label="Tech" value={dash.cell?.tech} />
              <DataField label="MCC/MNC" value={`${dash.cell?.mcc ?? "—"}/${dash.cell?.mnc ?? "—"}`} />
              <DataField label="Cell ID" value={dash.cell?.cell_id} />
              <DataField label="PCI" value={dash.cell?.pci} />
              <DataField label="Band" value={dash.cell?.band} />
              <DataField label="Bandwidth" value={dash.cell?.bandwidth} />
              {dash.cell?.scells && dash.cell.scells.length > 0 && (
                <DataField label="Aggregated"
                  value={dash.cell.scells.map((c) => c.band || "?").join(" + ")} />
              )}
            </>
          )}
        </Card>
        <Card title="System">
          {errs.sysinfo ? <div className="text-red-700 text-sm">{errs.sysinfo}</div> : (
            <>
              <DataField label="Hostname" value={dash.sysinfo?.hostname} />
              <DataField label="Uptime" value={formatUptime(dash.sysinfo?.uptime)} />
              <DataField label="Load" value={formatLoads(dash.sysinfo?.loads)} />
              {info?.data_path && <DataField label="Data path" value={info.data_path} />}
              {info?.usb_protocol && <DataField label="USB protocol" value={info.usb_protocol} />}
              {(dash.sysinfo?.interfaces ?? []).length > 0 && (
                <div className="mt-3 pt-3 border-t border-slate-100 space-y-2">
                  {(dash.sysinfo?.interfaces ?? []).map((ifc) => (
                    <div key={ifc.name}>
                      <div className="text-xs uppercase tracking-wide text-slate-500">{ifc.name}</div>
                      {(ifc.ipv4 ?? []).map((a) => (
                        <div key={a} className="font-mono text-sm text-slate-900">{a}</div>
                      ))}
                      {(ifc.ipv6 ?? []).map((a) => (
                        <div key={a} className="font-mono text-xs text-slate-500 break-all">{a}</div>
                      ))}
                    </div>
                  ))}
                </div>
              )}
            </>
          )}
        </Card>
      </div>
    </div>
  );
}
