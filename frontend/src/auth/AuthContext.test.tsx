import { describe, it, expect } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "../test/server";
import { AuthProvider } from "./AuthContext";
import { useAuth } from "../hooks/useAuth";

function Probe() {
  const { user, authenticated, loading } = useAuth();
  if (loading) return <span>loading</span>;
  return <span>{authenticated ? `hi ${user}` : "anon"}</span>;
}

describe("AuthContext", () => {
  it("loads whoami on mount", async () => {
    render(<AuthProvider><Probe /></AuthProvider>);
    await waitFor(() => expect(screen.getByText("hi admin")).toBeInTheDocument());
  });

  it("treats unauthenticated whoami as anonymous", async () => {
    server.use(
      http.get("/api/auth/whoami", () =>
        HttpResponse.json({ user: "", authenticated: false }),
      ),
    );
    render(<AuthProvider><Probe /></AuthProvider>);
    await waitFor(() => expect(screen.getByText("anon")).toBeInTheDocument());
  });
});
