import { describe, it, expect } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "../test/server";
import { Home } from "./Home";

describe("Home page", () => {
  it("renders sim+cell+signal from dashboard", async () => {
    server.use(http.get("/api/dashboard", () =>
      HttpResponse.json({
        sim: { slot: 1, imsi: "i", iccid: "c", operator: "T-Mobile", apn: "a", apn_ip: "1.1.1.1", registered: true },
        cell: { tech: "NR5G-SA", mcc: "310", mnc: "260", cell_id: "X", pci: 71, band: "n41", bandwidth: "100", signal: { rsrp: -98, rsrq: -12, sinr: 7, tech: "NR5G-SA" }, neighbors: [] },
        signal: { rsrp: -98, rsrq: -12, sinr: 7, tech: "NR5G-SA" },
        sysinfo: { hostname: "modem", uptime: 100, load: [0,0,0], mem_total: 1, mem_free: 1 },
        errors: {},
      })));
    render(<Home />);
    await waitFor(() => expect(screen.getByText("T-Mobile")).toBeInTheDocument());
    expect(screen.getAllByText("n41").length).toBeGreaterThan(0);
  });

  it("shows per-component errors", async () => {
    server.use(http.get("/api/dashboard", () =>
      HttpResponse.json({ sim: null, cell: null, signal: null, sysinfo: null, errors: { sim: "no sim" } })));
    render(<Home />);
    await waitFor(() => expect(screen.getByText(/no sim/i)).toBeInTheDocument());
  });
});
