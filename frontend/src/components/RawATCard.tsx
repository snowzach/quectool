import { useState } from "react";
import { Card } from "./Card";
import { atcmd } from "../api/atcmd";

const RAW_COMMANDS = [
  `AT+QENG="servingcell"`,
  `AT+QENG="neighbourcell"`,
  `AT+QCAINFO`,
];

export function RawATCard() {
  const [showRaw, setShowRaw] = useState(false);
  const [raw, setRaw] = useState<Record<string, string[]>>({});
  const [rawErr, setRawErr] = useState<string | null>(null);

  const fetchRaw = async () => {
    setRawErr(null);
    try {
      const out: Record<string, string[]> = {};
      for (const cmd of RAW_COMMANDS) {
        const r = await atcmd(cmd);
        out[cmd] = r.response ?? [];
      }
      setRaw(out);
    } catch (e: any) {
      setRawErr(e?.message ?? "fetch failed");
    }
  };

  return (
    <Card title="Raw AT Responses">
      <button
        onClick={() => {
          const next = !showRaw;
          setShowRaw(next);
          if (next && Object.keys(raw).length === 0) void fetchRaw();
        }}
        className="text-sm text-accent hover:text-accent-hover"
      >
        {showRaw ? "Hide" : "Show"} raw QENG / QCAINFO output
      </button>
      {showRaw && (
        <div className="mt-3 space-y-3">
          {rawErr && <div className="text-red-700 text-sm">{rawErr}</div>}
          {RAW_COMMANDS.map((cmd) => (
            <div key={cmd}>
              <div className="text-xs font-mono text-slate-500 mb-1">{cmd}</div>
              <pre className="bg-slate-100 rounded p-2 text-xs whitespace-pre-wrap font-mono">
                {raw[cmd] ? raw[cmd].join("\n") || "(empty)" : "loading…"}
              </pre>
            </div>
          ))}
          <button onClick={() => void fetchRaw()} className="text-xs text-slate-500 hover:text-slate-700">
            refresh
          </button>
        </div>
      )}
    </Card>
  );
}
