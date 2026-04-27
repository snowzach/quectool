import { ReactNode } from "react";
import { useLocation } from "wouter";
import { Navbar } from "./Navbar";

// Routes that want the full viewport width (no max-w-6xl chrome) — currently
// just the terminal, where every column counts.
const FULL_WIDTH_ROUTES = new Set(["/console"]);

export function Layout({ children }: { children: ReactNode }) {
  const [loc] = useLocation();
  const fullWidth = FULL_WIDTH_ROUTES.has(loc);
  return (
    <div className="h-full flex flex-col">
      <Navbar />
      <main className={fullWidth
        ? "flex-1 min-h-0"
        : "mx-auto max-w-6xl w-full px-4 py-6 flex-1"}>
        {children}
      </main>
      <img src="/gopher.png" alt=""
        className="hidden lg:block fixed bottom-0 right-2 w-24 opacity-60 pointer-events-none select-none z-0"
        aria-hidden="true" />
    </div>
  );
}
