import { describe, it, expect } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "../test/server";
import { AuthProvider } from "../auth/AuthContext";
import { Login } from "./Login";

function harness() {
  // whoami returns anon so AuthProvider doesn't auto-redirect.
  server.use(http.get("/api/auth/whoami", () => HttpResponse.json({ user: "", authenticated: false })));
  return render(<AuthProvider><Login /></AuthProvider>);
}

describe("Login page", () => {
  it("submits credentials", async () => {
    let captured: any = null;
    server.use(
      http.post("/api/auth/login", async ({ request }) => {
        captured = await request.json();
        return HttpResponse.json({ user: "admin", token: "t" });
      }),
    );
    harness();
    fireEvent.change(screen.getByLabelText("User"), { target: { value: "admin" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "password" } });
    fireEvent.click(screen.getByRole("button", { name: /log in/i }));
    await waitFor(() => expect(captured).toEqual({ user: "admin", password: "password" }));
  });

  it("shows error on bad credentials", async () => {
    server.use(http.post("/api/auth/login", () =>
      HttpResponse.json({ error: "invalid credentials", code: "ERR_BAD_CREDENTIALS" }, { status: 401 })));
    harness();
    fireEvent.change(screen.getByLabelText("User"), { target: { value: "x" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "y" } });
    fireEvent.click(screen.getByRole("button", { name: /log in/i }));
    await waitFor(() => expect(screen.getByText(/invalid credentials/i)).toBeInTheDocument());
  });
});
