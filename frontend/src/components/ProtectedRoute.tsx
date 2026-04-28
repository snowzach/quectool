import { ReactNode } from "react";
import { useLocation } from "wouter";
import { useAuth } from "../hooks/useAuth";

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { authenticated, loading } = useAuth();
  const [, navigate] = useLocation();
  if (loading) return <div className="p-6 text-slate-500">Loading…</div>;
  if (!authenticated) {
    queueMicrotask(() => navigate("/login"));
    return null;
  }
  return <>{children}</>;
}
