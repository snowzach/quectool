import { useContext } from "react";
import { AuthCtx } from "../auth/AuthContext";

export function useAuth() {
  const c = useContext(AuthCtx);
  if (!c) throw new Error("useAuth must be used inside AuthProvider");
  return c;
}
