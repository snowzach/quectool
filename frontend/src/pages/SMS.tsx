import { useState, FormEvent } from "react";
import { Card } from "../components/Card";
import { Spinner } from "../components/Spinner";
import { ToastStack } from "../components/Toast";
import { useApi } from "../hooks/useApi";
import { useToast } from "../hooks/useToast";
import { useDocumentTitle } from "../hooks/useDocumentTitle";
import { listSMS, sendSMS, deleteSMS, deleteSMSBulk, markAllReadSMS } from "../api/modem";

export function SMS() {
  useDocumentTitle("SMS");
  const { data, loading, error, reload } = useApi(listSMS);
  const [to, setTo] = useState("");
  const [body, setBody] = useState("");
  const [busy, setBusy] = useState(false);
  const toast = useToast();

  const onSend = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    try {
      await sendSMS(to, body);
      setTo(""); setBody("");
      toast.ok("sent");
      reload();
    } catch (e: any) {
      toast.error(e?.message ?? "send failed");
    } finally { setBusy(false); }
  };

  const onDelete = async (i: number) => {
    try { await deleteSMS(i); reload(); }
    catch (e: any) { toast.error(e?.message ?? "delete failed"); }
  };

  const unreadCount = (data ?? []).filter((m) => !m.read).length;
  const readCount = (data ?? []).filter((m) => m.read).length;
  const totalCount = (data ?? []).length;

  const onMarkAllRead = async () => {
    try { await markAllReadSMS(); toast.ok("marked read"); reload(); }
    catch (e: any) { toast.error(e?.message ?? "mark read failed"); }
  };
  const onDeleteRead = async () => {
    if (!confirm(`Delete ${readCount} read message${readCount === 1 ? "" : "s"}?`)) return;
    try { await deleteSMSBulk("read"); toast.ok("read messages deleted"); reload(); }
    catch (e: any) { toast.error(e?.message ?? "delete failed"); }
  };
  const onDeleteAll = async () => {
    if (!confirm(`Delete ALL ${totalCount} message${totalCount === 1 ? "" : "s"}? This cannot be undone.`)) return;
    try { await deleteSMSBulk("all"); toast.ok("inbox cleared"); reload(); }
    catch (e: any) { toast.error(e?.message ?? "delete failed"); }
  };

  return (
    <>
      <Card title="Send" className="mb-4">
        <form onSubmit={onSend} className="grid md:grid-cols-2 gap-x-4">
          <label className="block text-sm mb-2">
            <span className="block text-slate-600 mb-1">To</span>
            <input aria-label="To" className="w-full border rounded px-2 py-1" value={to} onChange={(e) => setTo(e.target.value)} />
          </label>
          <label className="block text-sm mb-2 md:col-span-2">
            <span className="block text-slate-600 mb-1">Body</span>
            <textarea aria-label="Body" className="w-full border rounded px-2 py-1" rows={2} value={body} onChange={(e) => setBody(e.target.value)} />
          </label>
          <div className="md:col-span-2">
            <button disabled={busy} className="bg-accent hover:bg-accent-hover text-white rounded px-4 py-1 disabled:opacity-50">
              {busy ? "Sending…" : "Send"}
            </button>
          </div>
        </form>
      </Card>

      <Card title="Inbox">
        {loading && <Spinner />}
        {error && <div className="text-red-700 text-sm">{error.message}</div>}
        {data && data.length === 0 && <div className="text-slate-500 text-sm">no messages</div>}
        {data && data.length > 0 && (
          <>
            <div className="flex items-center gap-2 mb-3 text-xs flex-wrap">
              <span className="text-slate-500">
                {totalCount} total
                {unreadCount > 0 && <span className="ml-1 text-accent font-medium">• {unreadCount} unread</span>}
              </span>
              <span className="ml-auto flex gap-2">
                <button disabled={unreadCount === 0} onClick={() => void onMarkAllRead()}
                  className="rounded px-2 py-1 bg-slate-100 hover:bg-slate-200 text-slate-700 disabled:opacity-40 disabled:hover:bg-slate-100">
                  Mark all read
                </button>
                <button disabled={readCount === 0} onClick={() => void onDeleteRead()}
                  className="rounded px-2 py-1 bg-amber-100 hover:bg-amber-200 text-amber-800 disabled:opacity-40 disabled:hover:bg-amber-100">
                  Delete read ({readCount})
                </button>
                <button disabled={totalCount === 0} onClick={() => void onDeleteAll()}
                  className="rounded px-2 py-1 bg-red-100 hover:bg-red-200 text-red-800 disabled:opacity-40 disabled:hover:bg-red-100">
                  Delete all
                </button>
              </span>
            </div>
            <ul className="divide-y divide-slate-100">
              {data.map((m) => (
                <li key={m.index} className="py-3 first:pt-0 last:pb-0">
                  <div className="flex items-start gap-3">
                    <div className="min-w-0 flex-1">
                      <div className="flex items-baseline gap-2 text-sm flex-wrap">
                        {!m.read && (
                          <span className="inline-block w-2 h-2 rounded-full bg-accent" aria-label="unread" />
                        )}
                        <span className={`break-all ${m.read ? "text-slate-700" : "font-semibold text-slate-900"}`}>
                          {m.from || "(no sender)"}
                        </span>
                        <span className="text-xs text-slate-500">{m.time}</span>
                        <span className="text-xs text-slate-400 font-mono">#{m.index}</span>
                      </div>
                      <div className="mt-1 text-sm text-slate-800 whitespace-pre-wrap break-words">
                        {m.body}
                      </div>
                    </div>
                    <button onClick={() => void onDelete(m.index)}
                      className="shrink-0 text-xs text-slate-500 hover:text-red-700 px-2 py-1 rounded hover:bg-red-50">
                      Delete
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          </>
        )}
      </Card>
      <ToastStack toasts={toast.toasts} />
    </>
  );
}
