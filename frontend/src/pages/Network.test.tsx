import { describe, it, expect } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "../test/server";
import { Network } from "./Network";

describe("Network page", () => {
  it("status strip 'Lock to current' button calls /cell-lock for 5G", async () => {
    let posted: any = null;
    server.use(
      http.post("/api/modem/cell-lock", async ({ request }) => {
        posted = await request.json();
        return new HttpResponse(null, { status: 204 });
      }),
    );
    render(<Network />);
    const btn = await screen.findByRole("button", { name: /lock 5g to current/i });
    fireEvent.click(btn);
    await waitFor(() => expect(posted).toEqual({ tech: "5g" }));
  });

  it("status strip Unlock button puts settings with lock_5g: false", async () => {
    server.use(
      http.get("/api/modem/settings", () =>
        HttpResponse.json({
          apn: { name: "", ip_type: "" },
          network: { mode: "AUTO" },
          bands: { lte: [], nr5g: [] },
          cell_lock: { lock_4g: false, lock_5g: true },
        })),
    );
    let put: any = null;
    server.use(http.put("/api/modem/settings", async ({ request }) => {
      put = await request.json();
      return HttpResponse.json({
        apn: { name: "", ip_type: "" },
        network: { mode: "AUTO" },
        bands: { lte: [], nr5g: [] },
        cell_lock: { lock_4g: false, lock_5g: false },
      });
    }));
    render(<Network />);
    const btn = await screen.findByRole("button", { name: /unlock 5g/i });
    fireEvent.click(btn);
    await waitFor(() => expect(put?.cell_lock?.lock_5g).toBe(false));
  });

  it("renders the advanced drawer collapsed", async () => {
    const { container } = render(<Network />);
    await screen.findByText(/advanced/i);
    const details = container.querySelector("details");
    expect(details).not.toBeNull();
    expect(details?.hasAttribute("open")).toBe(false);
  });
});
