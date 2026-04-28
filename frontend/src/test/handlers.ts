import { http, HttpResponse } from "msw";

// Default happy-path handlers covering every endpoint the SPA hits.
// Per-test overrides via server.use(http....) are how individual cases
// inject 401s, 502s, or specific payloads.
export const handlers = [
  http.get("/api/auth/whoami", () =>
    HttpResponse.json({ user: "admin", authenticated: true }),
  ),
  http.post("/api/auth/login", () =>
    HttpResponse.json({ user: "admin", token: "test-token" }),
  ),
  http.post("/api/auth/logout", () => new HttpResponse(null, { status: 204 })),

  http.get("/api/modem/info", () =>
    HttpResponse.json({
      manufacturer: "Quectel",
      model: "RM520N-GL",
      firmware: "RM520NGLAAR03A07M4G",
      imei: "000000000000000",
      capabilities: ["band_lock_lte", "band_lock_5g", "network_scan", "sms", "nr5g_sa"],
      available_bands: { lte: [1, 2, 3, 4, 5, 7, 8, 12, 13, 14, 17, 20, 25, 28, 41, 66, 71], nr5g: [2, 5, 25, 41, 66, 71, 77, 78] },
    }),
  ),
  http.get("/api/modem/sim", () =>
    HttpResponse.json({
      slot: 1, imsi: "310260000000000", iccid: "8901260000000000000",
      operator: "T-Mobile", apn: "fast.t-mobile.com", apn_ip: "10.0.0.1", registered: true,
    }),
  ),
  http.get("/api/modem/cell", () =>
    HttpResponse.json({
      tech: "NR5G-SA", mcc: "310", mnc: "260", cell_id: "B0FB00", pci: 71,
      band: "n41", bandwidth: "100",
      signal: { rsrp: -98, rsrq: -12, sinr: 7, tech: "NR5G-SA" },
      neighbors: [],
    }),
  ),
  http.get("/api/modem/signal", () =>
    HttpResponse.json({ rsrp: -98, rsrq: -12, sinr: 7, tech: "NR5G-SA" }),
  ),
  http.get("/api/modem/settings", () =>
    HttpResponse.json({
      apn: { name: "fast.t-mobile.com", ip_type: "IPV4V6" },
      network: { mode: "AUTO" },
      bands: { lte: [2, 4, 12, 71], nr5g: [41, 71] },
      cell_lock: { lock_4g: false, lock_5g: false },
    }),
  ),
  http.put("/api/modem/settings", () =>
    HttpResponse.json({
      apn: { name: "fast.t-mobile.com", ip_type: "IPV4V6" },
      network: { mode: "AUTO" },
      bands: { lte: [2, 4, 12, 71], nr5g: [41, 71] },
      cell_lock: { lock_4g: false, lock_5g: false },
    }),
  ),
  http.post("/api/modem/cell-lock", () => new HttpResponse(null, { status: 204 })),
  http.post("/api/modem/scan", () =>
    HttpResponse.json([
      { operator: "T-Mobile", mcc: "310", mnc: "260", tech: "NR5G", state: "current" },
      { operator: "AT&T", mcc: "310", mnc: "410", tech: "LTE", state: "available" },
    ]),
  ),
  http.get("/api/modem/sms", () =>
    HttpResponse.json([
      { index: 1, from: "+15555550100", time: "26/04/27,12:00:00-20", body: "hi", read: true },
    ]),
  ),
  http.post("/api/modem/sms", () => HttpResponse.json({ index: 2 })),
  http.delete("/api/modem/sms/:index", () => new HttpResponse(null, { status: 204 })),

  http.get("/api/atcmd", ({ request }) => {
    const url = new URL(request.url);
    const cmd = url.searchParams.get("cmd") ?? "";
    return HttpResponse.json({ command: cmd, response: "OK", lines: ["OK"] });
  }),
  http.get("/api/sysinfo", () =>
    HttpResponse.json({
      hostname: "modem", uptime: 1234, load: [0.1, 0.2, 0.3],
      mem_total: 524288000, mem_free: 100000000,
    }),
  ),
  http.get("/api/dashboard", () =>
    HttpResponse.json({
      sim: null, cell: null, signal: null, sysinfo: null, errors: {},
    }),
  ),
];
