import { OperatorScanCard } from "./OperatorScanCard";
import { ManualLockForm } from "./ManualLockForm";
import { NeighboursCard } from "./NeighboursCard";
import { RawATCard } from "./RawATCard";
import { Card } from "./Card";
import type { CellInfo } from "../types/modem";

export function NetworkAdvancedDrawer({
  cell, onManualLock,
}: {
  cell: CellInfo | null;
  onManualLock: (tech: "4g" | "5g", freq: number, pci: number, scs?: number, band?: number) => Promise<void>;
}) {
  return (
    <details className="rounded border border-slate-200 bg-white">
      <summary className="cursor-pointer px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50">
        Advanced — operator scan, manual lock, neighbours, raw AT
      </summary>
      <div className="p-3 space-y-4 border-t border-slate-100">
        <OperatorScanCard />
        <Card title="Manual Lock">
          <div className="text-xs text-slate-500 mb-2">
            Type EARFCN/ARFCN + PCI yourself. Use this when you know the exact
            cell parameters and don't want to run a survey.
          </div>
          <ManualLockForm onLock={onManualLock} />
        </Card>
        <NeighboursCard neighbors={cell?.neighbors} />
        <RawATCard />
      </div>
    </details>
  );
}
