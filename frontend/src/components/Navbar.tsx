import { Link, useLocation } from "wouter";
import { useAuth } from "../hooks/useAuth";

const links = [
  { href: "/", label: "Home" },
  { href: "/settings", label: "Settings" },
  { href: "/network", label: "Network" },
  { href: "/sms", label: "SMS" },
  { href: "/deviceinfo", label: "Device" },
  { href: "/atcmd", label: "AT" },
  { href: "/console", label: "Console" },
];

export function Navbar() {
  const [loc] = useLocation();
  const { user, logout } = useAuth();
  return (
    <nav className="bg-gradient-to-r from-indigo-700 via-blue-700 to-sky-700 shadow-md">
      <div className="mx-auto max-w-6xl px-4 py-2 flex items-center gap-4">
        <span className="font-bold tracking-tight text-white flex items-center gap-1.5">
          <span className="inline-block w-2 h-2 rounded-full bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.8)]" />
          quectool
        </span>
        <ul className="flex gap-1 text-sm">
          {links.map((l) => (
            <li key={l.href}>
              <Link href={l.href}
                className={`px-3 py-1.5 rounded-md transition-colors ${
                  loc === l.href
                    ? "bg-white/20 text-white font-medium"
                    : "text-blue-100 hover:bg-white/10 hover:text-white"
                }`}>
                {l.label}
              </Link>
            </li>
          ))}
        </ul>
        <div className="ml-auto flex items-center gap-3 text-sm text-blue-100">
          <Link href="/account"
            title="Account"
            className={`px-2 py-1 rounded hover:bg-white/10 hover:text-white transition-colors ${
              loc === "/account" ? "bg-white/20 text-white font-medium" : ""
            }`}>
            {user}
          </Link>
          <button onClick={() => void logout()}
            className="px-2 py-1 rounded hover:bg-white/10 hover:text-white transition-colors">Log out</button>
        </div>
      </div>
    </nav>
  );
}
