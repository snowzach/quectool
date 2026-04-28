import { Card } from "./Card";
import type { CellInfo } from "../types/modem";

export function NeighboursCard({ neighbors }: { neighbors: CellInfo[] | undefined }) {
  const list = neighbors ?? [];
  return (
    <Card title="Neighbour Cells">
      {list.length === 0 ? (
        <div className="text-slate-500 text-sm">
          No neighbours reported. The modem only populates this list during
          cell reselection — typically when idle or on weaker signal. NR5G-SA
          additionally has no standard neighbour query in the QENG family.
        </div>
      ) : (
        <table className="w-full text-sm">
          <thead className="text-slate-500"><tr><th className="text-left">Tech</th><th>PCI</th><th>Band</th><th>RSRP</th><th>RSRQ</th><th>SINR</th></tr></thead>
          <tbody>
            {list.map((n, i) => (
              <tr key={i} className="border-t border-slate-100">
                <td>{n.tech}</td>
                <td className="text-center">{n.pci}</td>
                <td className="text-center">{n.band || "—"}</td>
                <td className="text-center">{n.signal?.rsrp ?? "—"}</td>
                <td className="text-center">{n.signal?.rsrq ?? "—"}</td>
                <td className="text-center">{n.signal?.sinr ?? "—"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </Card>
  );
}
