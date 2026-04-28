import { createContext, useCallback, useEffect, useState, ReactNode } from "react";
import * as authApi from "../api/auth";

export interface AuthState {
  user: string;
  authenticated: boolean;
  loading: boolean;
  refresh: () => Promise<void>;
  login: (user: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

export const AuthCtx = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState("");
  const [authenticated, setAuthed] = useState(false);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    try {
      const w = await authApi.whoami();
      setUser(w.user);
      setAuthed(w.authenticated);
    } catch {
      setUser("");
      setAuthed(false);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void refresh(); }, [refresh]);

  const doLogin = useCallback(async (u: string, p: string) => {
    const r = await authApi.login(u, p);
    setUser(r.user);
    setAuthed(true);
  }, []);

  const doLogout = useCallback(async () => {
    try { await authApi.logout(); } finally {
      setUser("");
      setAuthed(false);
    }
  }, []);

  return (
    <AuthCtx.Provider value={{ user, authenticated, loading, refresh, login: doLogin, logout: doLogout }}>
      {children}
    </AuthCtx.Provider>
  );
}
