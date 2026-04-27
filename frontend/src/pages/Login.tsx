import { FormEvent, useState } from "react";
import { useLocation } from "wouter";
import { useAuth } from "../hooks/useAuth";
import { useDocumentTitle } from "../hooks/useDocumentTitle";

export function Login() {
  useDocumentTitle("Sign in");
  const { login, authenticated } = useAuth();
  const [, navigate] = useLocation();
  const [user, setUser] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  if (authenticated) {
    queueMicrotask(() => navigate("/"));
  }

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null); setBusy(true);
    try {
      await login(user, password);
      navigate("/");
    } catch (e: any) {
      setError(e?.message ?? "login failed");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-b from-slate-50 to-slate-100">
      <form onSubmit={onSubmit} className="relative bg-white border border-slate-200 rounded-lg shadow-md p-6 w-full max-w-sm">
        <h1 className="text-lg font-semibold mb-4">quectool</h1>
        <label className="block text-sm mb-3">
          <span className="block text-slate-600 mb-1">User</span>
          <input className="w-full border rounded px-2 py-1" value={user} onChange={(e) => setUser(e.target.value)} autoFocus />
        </label>
        <label className="block text-sm mb-4">
          <span className="block text-slate-600 mb-1">Password</span>
          <input type="password" className="w-full border rounded px-2 py-1" value={password} onChange={(e) => setPassword(e.target.value)} />
        </label>
        {error && <div className="mb-3 text-sm text-red-700">{error}</div>}
        <button disabled={busy} className="w-full bg-accent hover:bg-accent-hover text-white rounded py-2 disabled:opacity-50">
          {busy ? "Logging in…" : "Log in"}
        </button>
      </form>
    </div>
  );
}
