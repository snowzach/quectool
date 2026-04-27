import { useEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import { useDocumentTitle } from "../hooks/useDocumentTitle";

export function Console() {
  useDocumentTitle("Console");
  const containerRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;
    const term = new Terminal({ convertEol: false, cursorBlink: true });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(containerRef.current);
    fit.fit();
    const enc = new TextEncoder();

    // Current socket lives in this ref so the persistent term.onData /
    // term.onResize handlers always send to whichever connection is live.
    // On reconnect we just point this at the new socket.
    const sockRef: { current: WebSocket | null } = { current: null };
    let connected = false;

    const sendResize = () => {
      const s = sockRef.current;
      if (s && s.readyState === WebSocket.OPEN) {
        s.send(JSON.stringify({ cols: term.cols, rows: term.rows }));
      }
    };

    const connect = () => {
      const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
      const url = `${proto}//${window.location.host}/api/terminal`
        + `?LINES=${term.rows}&COLUMNS=${term.cols}&TERM=xterm-256color`;
      const sock = new WebSocket(url);
      sock.binaryType = "arraybuffer";
      sockRef.current = sock;

      sock.onopen = () => {
        connected = true;
        sendResize();
      };
      sock.onmessage = (ev) => {
        if (ev.data instanceof ArrayBuffer) term.write(new Uint8Array(ev.data));
        else term.write(ev.data);
      };
      sock.onclose = () => {
        connected = false;
        term.write("\r\n\x1b[33m[connection closed — press Enter to reconnect]\x1b[0m\r\n");
      };
      sock.onerror = () => {
        // The close event will fire right after; let it print the message.
      };
    };

    connect();

    // Forward keystrokes to the live socket. When disconnected, intercept
    // Enter (CR) to start a new session instead of writing into the void.
    const dataDisp = term.onData((d) => {
      if (!connected) {
        // Carriage return = Enter on most terminals.
        if (d === "\r" || d === "\n") {
          term.write("\r\n\x1b[36m[reconnecting…]\x1b[0m\r\n");
          connect();
        }
        return;
      }
      sockRef.current?.send(enc.encode(d));
    });
    const resDisp = term.onResize(sendResize);

    // Watch the container so we refit when the slot becomes visible after
    // being display:none (e.g. user came back from another tab). fit.fit()
    // throws on zero-size containers — guard with offset checks.
    const ro = new ResizeObserver(() => {
      const el = containerRef.current;
      if (el && el.offsetWidth > 0 && el.offsetHeight > 0) {
        try { fit.fit(); } catch { /* zero-size flicker; next tick will retry */ }
      }
    });
    ro.observe(containerRef.current);

    return () => {
      ro.disconnect();
      dataDisp.dispose();
      resDisp.dispose();
      sockRef.current?.close();
      term.dispose();
    };
  }, []);

  return <div ref={containerRef} className="h-full w-full bg-black overflow-hidden" />;
}
