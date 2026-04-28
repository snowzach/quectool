import { ReactNode } from "react";
export function Card({ title, children, className = "" }: { title?: string; children: ReactNode; className?: string }) {
  return (
    <section className={`bg-white border border-slate-200 rounded-lg p-4 shadow-sm ${className}`}>
      {title && (
        <h2 className="text-sm font-semibold text-slate-700 mb-3 pb-2 border-b border-slate-100 flex items-center gap-2">
          <span className="inline-block w-1 h-4 rounded-full bg-gradient-to-b from-indigo-500 to-sky-500" />
          {title}
        </h2>
      )}
      {children}
    </section>
  );
}
