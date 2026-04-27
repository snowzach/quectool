import { describe, it, expect } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "../test/server";
import { useApi } from "./useApi";
import { apiFetch } from "../api/client";

function Probe({ path }: { path: string }) {
  const { data, error, loading } = useApi<{ msg: string }>(() => apiFetch<{ msg: string }>(path), [path]);
  if (loading) return <span>loading</span>;
  if (error) return <span>err:{error.message}</span>;
  return <span>{data?.msg}</span>;
}

describe("useApi", () => {
  it("loads and renders data", async () => {
    server.use(http.get("/api/x", () => HttpResponse.json({ msg: "ok" })));
    render(<Probe path="/api/x" />);
    await waitFor(() => expect(screen.getByText("ok")).toBeInTheDocument());
  });

  it("captures errors", async () => {
    server.use(http.get("/api/y", () =>
      HttpResponse.json({ error: "bad", code: "ERR" }, { status: 502 })));
    render(<Probe path="/api/y" />);
    await waitFor(() => expect(screen.getByText("err:bad")).toBeInTheDocument());
  });
});
