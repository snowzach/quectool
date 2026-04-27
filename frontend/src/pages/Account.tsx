import { FormEvent, useState } from "react";
import { Card } from "../components/Card";
import { useAuth } from "../hooks/useAuth";
import { useDocumentTitle } from "../hooks/useDocumentTitle";
import { changePassword } from "../api/auth";

export function Account() {
  useDocumentTitle("Account");
  const { user } = useAuth();
  const [oldPw, setOldPw] = useState("");
  const [newPw, setNewPw] = useState("");
  const [confirmPw, setConfirmPw] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [ok, setOk] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null); setOk(false);
    if (newPw !== confirmPw) {
      setError("New passwords do not match");
      return;
    }
    setBusy(true);
    try {
      await changePassword(oldPw, newPw);
      setOldPw(""); setNewPw(""); setConfirmPw("");
      setOk(true);
    } catch (e: any) {
      setError(e?.message ?? "could not change password");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="max-w-md mx-auto space-y-4">
      <Card title="Account">
        <div className="text-sm text-slate-600">
          Logged in as <span className="font-mono font-medium text-slate-900">{user || "—"}</span>.
        </div>
      </Card>

      <Card title="Change Password">
        <div className="text-xs text-slate-500 mb-3">
          Updates the bcrypt hash in the credentials file. Takes effect immediately;
          existing sessions stay valid until they expire.
        </div>
        <form onSubmit={submit} className="space-y-2">
          <label className="block text-sm">
            <span className="block text-slate-600 mb-1">Current password</span>
            <input type="password" value={oldPw} onChange={(e) => setOldPw(e.target.value)}
              autoComplete="current-password" required
              className="w-full border rounded px-2 py-1" />
          </label>
          <label className="block text-sm">
            <span className="block text-slate-600 mb-1">New password</span>
            <input type="password" value={newPw} onChange={(e) => setNewPw(e.target.value)}
              autoComplete="new-password" required minLength={6}
              className="w-full border rounded px-2 py-1" />
          </label>
          <label className="block text-sm">
            <span className="block text-slate-600 mb-1">Confirm new password</span>
            <input type="password" value={confirmPw} onChange={(e) => setConfirmPw(e.target.value)}
              autoComplete="new-password" required minLength={6}
              className="w-full border rounded px-2 py-1" />
          </label>
          {error && <div className="text-sm text-red-700">{error}</div>}
          {ok && <div className="text-sm text-emerald-700">Password changed.</div>}
          <button type="submit" disabled={busy || !oldPw || !newPw}
            className="bg-accent hover:bg-accent-hover text-white rounded px-4 py-1 disabled:opacity-50">
            {busy ? "Saving…" : "Change password"}
          </button>
        </form>
      </Card>
    </div>
  );
}
