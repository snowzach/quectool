import { useEffect, useState } from "react";
import { Card } from "../components/Card";
import { Spinner } from "../components/Spinner";
import { ToastStack } from "../components/Toast";
import { useApi } from "../hooks/useApi";
import { useToast } from "../hooks/useToast";
import { useDocumentTitle } from "../hooks/useDocumentTitle";
import { getInfo, getSettings, putSettings, rebootModem, setDataPath } from "../api/modem";
import type {
  Settings as S, APNSettings, NetworkSettings, BandSet,
} from "../types/modem";

const NETWORK_MODES: Array<{ value: string; label: string; help: string }> = [
  { value: "AUTO", label: "Auto", help: "Modem picks best available — LTE, 5G NSA, or 5G SA." },
  { value: "LTE_ONLY", label: "LTE only", help: "Force 4G — no 5G even if available." },
  { value: "NR5G_NSA_ONLY", label: "LTE (with 5G NSA)", help: "Primary connection is LTE; modem aggregates a 5G secondary carrier when one is in range. Falls back to plain LTE otherwise — you'll often see LTE on the dashboard." },
  { value: "NR5G_SA_ONLY", label: "5G SA", help: "5G Standalone — no LTE fallback. Requires a 5G SA cell in range or you'll be offline." },
];

const IP_TYPES = ["IPV4", "IPV6", "IPV4V6"];

function APNCard({ initial, onSave }: { initial: APNSettings; onSave: (a: APNSettings) => Promise<void> }) {
  const [name, setName] = useState(initial.name);
  const [ipType, setIPType] = useState(initial.ip_type || "IPV4V6");
  const [busy, setBusy] = useState(false);
  useEffect(() => { setName(initial.name); setIPType(initial.ip_type || "IPV4V6"); }, [initial.name, initial.ip_type]);
  const dirty = name !== initial.name || ipType !== (initial.ip_type || "IPV4V6");

  return (
    <Card title="APN">
      <div className="grid md:grid-cols-2 gap-4 mb-3">
        <label className="block text-sm">
          <span className="block text-slate-600 mb-1">APN name</span>
          <input aria-label="APN name" className="w-full border rounded px-2 py-1 font-mono"
            value={name} onChange={(e) => setName(e.target.value)} />
        </label>
        <label className="block text-sm">
          <span className="block text-slate-600 mb-1">IP type</span>
          <select aria-label="IP type" className="w-full border rounded px-2 py-1"
            value={ipType} onChange={(e) => setIPType(e.target.value)}>
            {IP_TYPES.map((v) => <option key={v} value={v}>{v}</option>)}
          </select>
        </label>
      </div>
      <button disabled={!dirty || busy}
        onClick={async () => { setBusy(true); try { await onSave({ name, ip_type: ipType }); } finally { setBusy(false); } }}
        className="bg-accent hover:bg-accent-hover text-white rounded px-4 py-1 disabled:opacity-50">
        {busy ? "Saving…" : "Save APN"}
      </button>
    </Card>
  );
}

function NetworkCard({ initial, onSave }: { initial: NetworkSettings; onSave: (n: NetworkSettings) => Promise<void> }) {
  const [mode, setMode] = useState(initial.mode);
  const [busy, setBusy] = useState(false);
  useEffect(() => { setMode(initial.mode); }, [initial.mode]);
  const dirty = mode !== initial.mode;

  return (
    <Card title="Network Mode">
      {initial.mode === "CUSTOM" && (
        <div className="mb-3 text-sm text-amber-700 bg-amber-50 border border-amber-200 rounded p-2">
          Modem is in a custom mode (mode_pref + nr5g_disable_mode combination not in the standard set).
          Saving will normalize it.
        </div>
      )}
      <div className="space-y-2 mb-3">
        {NETWORK_MODES.map((m) => (
          <label key={m.value} className="flex items-start gap-2 text-sm cursor-pointer">
            <input type="radio" name="netmode" value={m.value} checked={mode === m.value}
              aria-label={m.label}
              onChange={() => setMode(m.value)} className="mt-1" />
            <div>
              <div className="font-medium">{m.label}</div>
              <div className="text-slate-500 text-xs">{m.help}</div>
            </div>
          </label>
        ))}
      </div>
      <button disabled={!dirty || busy}
        onClick={async () => { setBusy(true); try { await onSave({ mode }); } finally { setBusy(false); } }}
        className="bg-accent hover:bg-accent-hover text-white rounded px-4 py-1 disabled:opacity-50">
        {busy ? "Saving…" : "Save Network Mode"}
      </button>
    </Card>
  );
}

function BandPicker({ title, available, selected, onChange }: {
  title: string;
  available: number[];
  selected: number[];
  onChange: (next: number[]) => void;
}) {
  const all = new Set(selected);
  const toggle = (b: number) => {
    const next = new Set(all);
    if (next.has(b)) next.delete(b); else next.add(b);
    onChange(Array.from(next).sort((a, b) => a - b));
  };
  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm font-medium text-slate-700">{title}</span>
        <div className="flex gap-2 text-xs">
          <button type="button" onClick={() => onChange([...available])}
            className="text-accent hover:text-accent-hover">All</button>
          <button type="button" onClick={() => onChange([])}
            className="text-slate-500 hover:text-slate-700">None</button>
        </div>
      </div>
      <div className="grid grid-cols-4 sm:grid-cols-6 gap-2">
        {available.map((b) => (
          <label key={b} className={`flex items-center gap-1 px-2 py-1 rounded border text-xs cursor-pointer ${
            all.has(b) ? "bg-accent/10 border-accent text-accent" : "bg-white border-slate-200 text-slate-700 hover:border-slate-300"
          }`}>
            <input type="checkbox" checked={all.has(b)} onChange={() => toggle(b)} className="hidden" />
            <span className="font-mono">{title.startsWith("NR") ? "n" : "B"}{b}</span>
          </label>
        ))}
      </div>
    </div>
  );
}

function BandsCard({ initial, available, onSave }: {
  initial: BandSet;
  available: BandSet;
  onSave: (b: BandSet) => Promise<void>;
}) {
  const [lte, setLTE] = useState<number[]>(initial.lte ?? []);
  const [nr5g, setNR5G] = useState<number[]>(initial.nr5g ?? []);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    setLTE(initial.lte ?? []);
    setNR5G(initial.nr5g ?? []);
  }, [initial.lte, initial.nr5g]);

  const same = (a: number[], b: number[]) => a.length === b.length && a.every((x, i) => x === b[i]);
  const dirty = !same(lte, initial.lte ?? []) || !same(nr5g, initial.nr5g ?? []);
  const empty = lte.length === 0 && nr5g.length === 0;

  return (
    <Card title="Preferred Bands">
      <div className="text-xs text-slate-500 mb-3">
        Which bands the modem may use. Click <span className="font-medium">All</span> to enable
        every band the radio supports, or pick specific ones. Saving with nothing checked
        will disable the radio — don't do that.
      </div>
      {empty && (
        <div className="mb-3 text-sm text-amber-700 bg-amber-50 border border-amber-200 rounded p-2">
          No bands enabled — the modem will not connect. Pick at least one or click All.
        </div>
      )}
      <div className="space-y-4 mb-3">
        <BandPicker title="LTE" available={available.lte ?? []} selected={lte} onChange={setLTE} />
        <BandPicker title="NR5G" available={available.nr5g ?? []} selected={nr5g} onChange={setNR5G} />
      </div>
      <button disabled={!dirty || busy}
        onClick={async () => {
          setBusy(true);
          try { await onSave({ lte, nr5g }); }
          finally { setBusy(false); }
        }}
        className="bg-accent hover:bg-accent-hover text-white rounded px-4 py-1 disabled:opacity-50">
        {busy ? "Saving…" : "Save Bands"}
      </button>
    </Card>
  );
}

export function Settings() {
  useDocumentTitle("Settings");
  const settings = useApi(getSettings);
  const info = useApi(getInfo);
  const toast = useToast();

  if (settings.loading || info.loading) return <Spinner />;
  if (settings.error) return <div className="text-red-700">{settings.error.message}</div>;
  if (info.error) return <div className="text-red-700">{info.error.message}</div>;
  if (!settings.data || !info.data) return null;

  const data = settings.data;
  const available = info.data.available_bands ?? { lte: [], nr5g: [] };

  const save = async (label: string, partial: Partial<S>) => {
    try {
      await putSettings(partial);
      toast.ok(`${label} saved`);
      settings.reload();
    } catch (e: any) {
      toast.error(e?.message ?? "save failed");
    }
  };

  return (
    <div className="space-y-4">
      <APNCard initial={data.apn} onSave={(apn) => save("APN", { apn })} />
      <NetworkCard initial={data.network} onSave={(network) => save("Network mode", { network })} />
      <BandsCard initial={data.bands} available={available} onSave={(bands) => save("Bands", { bands })} />
      <DataPathCard
        current={info.data.data_path}
        onChange={async (path) => {
          try {
            await setDataPath(path);
            toast.ok(`Data path set to ${path} — reboot the modem to apply`);
          } catch (e: any) {
            toast.error(e?.message ?? "set failed");
          }
        }}
      />
      <RebootCard onReboot={async () => {
        try {
          await rebootModem();
          toast.ok("Modem rebooting — back in ~20s");
        } catch (e: any) {
          toast.error(e?.message ?? "reboot failed");
        }
      }} />
      <ToastStack toasts={toast.toasts} />
    </div>
  );
}

function DataPathCard({ current, onChange }: {
  current: string | undefined;
  onChange: (path: "USB" | "PCIe") => Promise<void>;
}) {
  const [pending, setPending] = useState<"USB" | "PCIe" | null>(null);
  const [busy, setBusy] = useState(false);
  if (!current) return null;

  const desc: Record<string, string> = {
    USB: "Modem data flows over the M.2 USB interface — host sees it as a USB net device (qmi_wwan / cdc_ether / cdc_mbim depending on USB protocol).",
    PCIe: "Modem data flows over the PCIe interface — typically wired to a Realtek ethernet bridge on the carrier board, exposed as an ethN interface.",
  };

  return (
    <Card title="Data Path">
      <div className="text-xs text-slate-500 mb-3">
        Which physical interface carries the modem's data plane. Changing this
        only takes effect after a modem reboot, and may make the modem invisible
        to the host until the matching driver is in place.
      </div>
      <div className="space-y-2 mb-3">
        {(["USB", "PCIe"] as const).map((p) => {
          const active = current === p;
          return (
            <label key={p} className="flex items-start gap-2 text-sm cursor-pointer">
              <input type="radio" name="datapath" value={p}
                checked={pending ? pending === p : active}
                onChange={() => setPending(p)}
                className="mt-1" />
              <div>
                <div className="font-medium">
                  {p}{active && <span className="ml-2 text-xs text-emerald-700">(current)</span>}
                </div>
                <div className="text-slate-500 text-xs">{desc[p]}</div>
              </div>
            </label>
          );
        })}
      </div>
      {pending && pending !== current && (
        <div className="mb-3 text-sm text-amber-700 bg-amber-50 border border-amber-200 rounded p-2">
          Switching to <strong>{pending}</strong>. After saving, click "Reboot Modem"
          below for the change to take effect. If the host doesn't have the matching
          driver loaded for {pending}, the modem will not appear until you fix that.
        </div>
      )}
      <button disabled={!pending || pending === current || busy}
        onClick={async () => {
          if (!pending) return;
          setBusy(true);
          try { await onChange(pending); } finally { setBusy(false); }
        }}
        className="bg-accent hover:bg-accent-hover text-white rounded px-4 py-1 disabled:opacity-50">
        {busy ? "Saving…" : "Save Data Path"}
      </button>
    </Card>
  );
}

function RebootCard({ onReboot }: { onReboot: () => Promise<void> }) {
  const [busy, setBusy] = useState(false);
  const [confirming, setConfirming] = useState(false);
  return (
    <Card title="Reboot Modem">
      <div className="text-xs text-slate-500 mb-3">
        Full modem reset (AT+CFUN=1,1). The device drops off USB for ~15-20 seconds before
        coming back. Use this if the radio is wedged after a config change.
      </div>
      {confirming ? (
        <div className="flex items-center gap-2">
          <span className="text-sm text-amber-700">Are you sure?</span>
          <button disabled={busy}
            onClick={async () => { setBusy(true); try { await onReboot(); } finally { setBusy(false); setConfirming(false); } }}
            className="bg-red-600 hover:bg-red-700 text-white rounded px-3 py-1 text-sm disabled:opacity-50">
            {busy ? "Rebooting…" : "Yes, reboot"}
          </button>
          <button onClick={() => setConfirming(false)} disabled={busy}
            className="bg-slate-100 hover:bg-slate-200 text-slate-700 rounded px-3 py-1 text-sm">
            Cancel
          </button>
        </div>
      ) : (
        <button onClick={() => setConfirming(true)}
          className="bg-red-600 hover:bg-red-700 text-white rounded px-4 py-1">
          Reboot Modem
        </button>
      )}
    </Card>
  );
}
