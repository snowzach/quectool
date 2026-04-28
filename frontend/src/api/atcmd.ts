import { apiFetch } from "./client";
import type { ATResponse } from "../types/atserver";

export const atcmd = (cmd: string) =>
  apiFetch<ATResponse>(`/api/atcmd?atcmd=${encodeURIComponent(cmd)}`);
export type { ATResponse };
