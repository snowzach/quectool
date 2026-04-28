import type { ReactNode } from "react";

export function LockConfirmModal({
  title, body, confirmLabel, onConfirm, onCancel,
}: {
  title: string;
  body: ReactNode;
  confirmLabel: string;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4"
      onClick={(e) => { if (e.target === e.currentTarget) onCancel(); }}
    >
      <div className="bg-white rounded-lg shadow-lg max-w-md w-full p-5">
        <h3 className="text-lg font-semibold mb-2">{title}</h3>
        <div className="text-sm text-slate-700 mb-4">{body}</div>
        <div className="flex justify-end gap-2">
          <button
            type="button"
            onClick={onCancel}
            className="bg-slate-100 hover:bg-slate-200 text-slate-700 rounded px-3 py-1 text-sm"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onConfirm}
            className="bg-accent hover:bg-accent-hover text-white rounded px-3 py-1 text-sm"
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
