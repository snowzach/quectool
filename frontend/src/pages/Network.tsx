import { ToastStack } from "../components/Toast";
import { Spinner } from "../components/Spinner";
import { NetworkStatusStrip } from "../components/NetworkStatusStrip";
import { CellSurveyPanel } from "../components/CellSurveyPanel";
import { NetworkAdvancedDrawer } from "../components/NetworkAdvancedDrawer";
import { useApi } from "../hooks/useApi";
import { usePolling } from "../hooks/usePolling";
import { useToast } from "../hooks/useToast";
import { useDocumentTitle } from "../hooks/useDocumentTitle";
import {
  getCell, getSettings, putSettings, lockCell, lockCurrentCell,
} from "../api/modem";

export function Network() {
  useDocumentTitle("Network");
  const toast = useToast();
  const settings = useApi(getSettings);
  const { data: cell } = usePolling(getCell, 5_000);

  const onLockCurrent = async (tech: "4g" | "5g") => {
    try {
      await lockCurrentCell(tech);
      toast.ok(`${tech.toUpperCase()} locked to current cell`);
      settings.reload();
    } catch (e: any) { toast.error(e?.message ?? "lock failed"); }
  };

  const onUnlock = async (tech: "4g" | "5g") => {
    try {
      await putSettings({ cell_lock: {
        ...settings.data!.cell_lock,
        [tech === "4g" ? "lock_4g" : "lock_5g"]: false,
      } });
      toast.ok(`${tech.toUpperCase()} unlocked`);
      settings.reload();
    } catch (e: any) { toast.error(e?.message ?? "unlock failed"); }
  };

  const onSurveyLock = async (a: { tech: "4g" | "5g"; freq: number; pci: number; scs?: number; band?: number }) => {
    await lockCell(a.tech, a.freq, a.pci, a.scs, a.band);
    settings.reload();
  };

  const onManualLock = async (tech: "4g" | "5g", freq: number, pci: number, scs?: number, band?: number) => {
    try {
      await lockCell(tech, freq, pci, scs, band);
      toast.ok(`${tech.toUpperCase()} locked manually`);
      settings.reload();
    } catch (e: any) { toast.error(e?.message ?? "lock failed"); }
  };

  if (settings.loading) return <Spinner />;

  return (
    <div className="space-y-4">
      <NetworkStatusStrip
        cell={cell ?? null}
        cellLock={settings.data?.cell_lock ?? { lock_4g: false, lock_5g: false }}
        onLockCurrent={onLockCurrent}
        onUnlock={onUnlock}
      />
      <CellSurveyPanel
        currentCell={cell ?? null}
        onLock={onSurveyLock}
        onError={toast.error}
        onSuccess={toast.ok}
      />
      <NetworkAdvancedDrawer cell={cell ?? null} onManualLock={onManualLock} />
      <ToastStack toasts={toast.toasts} />
    </div>
  );
}
