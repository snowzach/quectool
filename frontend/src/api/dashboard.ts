import { apiFetch } from "./client";
import type { SimInfo, CellInfo, Signal } from "../types/modem";
import type { SysInfo } from "./sysinfo";

export interface Dashboard {
  sim: SimInfo | null;
  cell: CellInfo | null;
  signal: Signal | null;
  sysinfo: SysInfo | null;
  errors: Record<string, string>;
}
export const getDashboard = () => apiFetch<Dashboard>("/api/dashboard");
