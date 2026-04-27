import { describe, it, expect } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "../test/server";
import { Settings } from "./Settings";

describe("Settings page", () => {
  it("submits APN-only update", async () => {
    let captured: any = null;
    server.use(
      http.put("/api/modem/settings", async ({ request }) => {
        captured = await request.json();
        return HttpResponse.json({
          apn: { name: "new.apn", ip_type: "IPV4V6" },
          network: { mode: "AUTO" },
          bands: { lte: [2, 4, 12, 71], nr5g: [41, 71] },
          cell_lock: { lock_4g: false, lock_5g: false },
        });
      }),
    );
    render(<Settings />);
    await waitFor(() => screen.getByDisplayValue("fast.t-mobile.com"));
    fireEvent.change(screen.getByLabelText("APN name"), { target: { value: "new.apn" } });
    fireEvent.click(screen.getByRole("button", { name: /save apn/i }));
    await waitFor(() => expect(captured).toEqual({
      apn: { name: "new.apn", ip_type: "IPV4V6" },
    }));
  });

  it("submits network-mode-only update", async () => {
    let captured: any = null;
    server.use(
      http.put("/api/modem/settings", async ({ request }) => {
        captured = await request.json();
        return HttpResponse.json({
          apn: { name: "fast.t-mobile.com", ip_type: "IPV4V6" },
          network: { mode: "NR5G_SA_ONLY" },
          bands: { lte: [2, 4, 12, 71], nr5g: [41, 71] },
          cell_lock: { lock_4g: false, lock_5g: false },
        });
      }),
    );
    render(<Settings />);
    await waitFor(() => screen.getByLabelText("APN name"));
    fireEvent.click(screen.getByLabelText("5G SA"));
    fireEvent.click(screen.getByRole("button", { name: /save network mode/i }));
    await waitFor(() => expect(captured).toEqual({ network: { mode: "NR5G_SA_ONLY" } }));
  });

});
