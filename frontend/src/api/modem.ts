import { apiFetch } from "./client";
import type {
  Info, CellInfo, Signal, Settings, SettingsUpdate, ScanResult, SMS, CellSurveyResult,
} from "../types/modem";

export const getInfo     = () => apiFetch<Info>("/api/modem/info");
export const getCell     = () => apiFetch<CellInfo>("/api/modem/cell");
export const getSignal   = () => apiFetch<Signal>("/api/modem/signal");
export const getSettings = () => apiFetch<Settings>("/api/modem/settings");

// Mode/band changes trigger a CFUN cycle on the backend (~10s); allow 30s.
export const putSettings = (u: SettingsUpdate) =>
  apiFetch<Settings>("/api/modem/settings", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(u),
    timeoutMs: 30_000,
  });

// Scans take ~30s; allow 60s before aborting.
export const scan = () =>
  apiFetch<ScanResult[]>("/api/modem/scan", { method: "POST", timeoutMs: 60_000 });

// QSCAN sweeps GSM/LTE/NR5G across all bands; budget 2 minutes.
export const cellSurvey = () =>
  apiFetch<CellSurveyResult[]>("/api/modem/cell-survey", { method: "POST", timeoutMs: 130_000 });

export const listSMS = () => apiFetch<SMS[]>("/api/modem/sms");
export const sendSMS = (to: string, body: string) =>
  apiFetch("/api/modem/sms", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ to, body }),
  });
export const deleteSMS = (index: number) =>
  apiFetch(`/api/modem/sms/${index}`, { method: "DELETE" });

export const deleteSMSBulk = (scope: "read" | "all") =>
  apiFetch(`/api/modem/sms?scope=${scope}`, { method: "DELETE" });

export const markAllReadSMS = () =>
  apiFetch("/api/modem/sms/mark-all-read", { method: "POST" });

// Reboot returns immediately after the modem ACKs CFUN=1,1; the device then
// drops off USB for ~15-20s before coming back.
export const rebootModem = () =>
  apiFetch("/api/modem/reboot", { method: "POST" });

export const setDataPath = (path: "USB" | "PCIe") =>
  apiFetch("/api/modem/data-path", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path }),
  });

export const lockCurrentCell = (tech: "4g" | "5g") =>
  apiFetch("/api/modem/cell-lock", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ tech }),
  });

// Lock to an explicit cell. freq is EARFCN (LTE) / ARFCN (NR5G). scs is the
// QENG/QSCAN subcarrier-spacing index (0=15kHz, 1=30kHz, 2=60kHz, 3=120kHz);
// LTE ignores it. band is the numeric band number — required for 5G,
// ignored for LTE.
export const lockCell = (
  tech: "4g" | "5g", freq: number, pci: number, scs?: number, band?: number,
) =>
  apiFetch("/api/modem/cell-lock", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ tech, freq, pci, scs: scs ?? 0, band: band ?? 0 }),
  });
