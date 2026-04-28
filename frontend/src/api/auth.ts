import { apiFetch } from "./client";

export interface Whoami { user: string; authenticated: boolean }
export interface LoginResponse { user: string; token: string }

export const whoami = () =>
  apiFetch<Whoami>("/api/auth/whoami", { redirectOn401: false });

export const login = (user: string, password: string) =>
  apiFetch<LoginResponse>("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ user, password }),
    redirectOn401: false,
  });

export const logout = () =>
  apiFetch("/api/auth/logout", { method: "POST", redirectOn401: false });

export const changePassword = (oldPw: string, newPw: string) =>
  apiFetch("/api/auth/passwd", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ old: oldPw, new: newPw }),
    // 401 here means "the old password you typed is wrong" — NOT "your
    // session is bad". Without this, the global redirect-to-login on 401
    // swallows the error before the form's catch block sees it.
    redirectOn401: false,
  });
