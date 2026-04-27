// @vitest-environment node
// jsdom's AbortSignal is a different class from Node's, so MSW's fetch
// interceptor rejects it with "Expected signal to be an instance of AbortSignal".
// Switching to the node environment makes all AbortSignal instances come from
// the same realm and the integration works correctly.
// In node environment, fetch requires absolute URLs; we prefix with
// http://localhost so paths like /api/x resolve correctly.
import { describe, it, expect } from "vitest";
import { http, HttpResponse } from "msw";
import { server } from "../test/server";
import { apiFetch, ApiError } from "./client";

const BASE = "http://localhost";

describe("apiFetch", () => {
  it("returns parsed JSON on 2xx", async () => {
    server.use(http.get(`${BASE}/api/x`, () => HttpResponse.json({ a: 1 })));
    const got = await apiFetch<{ a: number }>(`${BASE}/api/x`);
    expect(got.a).toBe(1);
  });

  it("throws ApiError with code+message on error envelope", async () => {
    server.use(
      http.get(`${BASE}/api/x`, () =>
        HttpResponse.json({ error: "nope", code: "ERR_NOPE" }, { status: 502 }),
      ),
    );
    await expect(apiFetch(`${BASE}/api/x`)).rejects.toMatchObject({
      status: 502,
      code: "ERR_NOPE",
      message: "nope",
    });
    expect(await apiFetch(`${BASE}/api/x`).catch((e) => e)).toBeInstanceOf(ApiError);
  });

  it("returns null on 204", async () => {
    server.use(http.delete(`${BASE}/api/x`, () => new HttpResponse(null, { status: 204 })));
    const got = await apiFetch(`${BASE}/api/x`, { method: "DELETE" });
    expect(got).toBeNull();
  });

  it("aborts on timeout", async () => {
    server.use(
      http.get(`${BASE}/api/slow`, async () => {
        await new Promise((r) => setTimeout(r, 500));
        return HttpResponse.json({});
      }),
    );
    await expect(apiFetch(`${BASE}/api/slow`, { timeoutMs: 20 })).rejects.toThrow();
  });
});
