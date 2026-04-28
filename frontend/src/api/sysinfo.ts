import { apiFetch } from "./client";
import type { SysInfo } from "../types/sysinfo";

export const getSysInfo = () => apiFetch<SysInfo>("/api/sysinfo");
export type { SysInfo };
