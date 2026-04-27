import { describe, it, expect } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "../test/server";
import { AuthProvider } from "../auth/AuthContext";
import { Account } from "./Account";

function harness() {
  // whoami stub so AuthProvider settles to authenticated=true.
  server.use(http.get("/api/auth/whoami", () =>
    HttpResponse.json({ user: "admin", authenticated: true })));
  return render(<AuthProvider><Account /></AuthProvider>);
}

describe("Account page", () => {
  it("submits a password change", async () => {
    let captured: any = null;
    server.use(
      http.post("/api/auth/passwd", async ({ request }) => {
        captured = await request.json();
        return new HttpResponse(null, { status: 204 });
      }),
    );
    harness();
    await waitFor(() => screen.getByLabelText("Current password"));
    fireEvent.change(screen.getByLabelText("Current password"), { target: { value: "oldpw" } });
    fireEvent.change(screen.getByLabelText("New password"), { target: { value: "newpw1" } });
    fireEvent.change(screen.getByLabelText("Confirm new password"), { target: { value: "newpw1" } });
    fireEvent.click(screen.getByRole("button", { name: /change password/i }));
    await waitFor(() => expect(captured).toEqual({ old: "oldpw", new: "newpw1" }));
  });

  it("renders the server error inline when old password is wrong", async () => {
    server.use(
      http.post("/api/auth/passwd", () =>
        HttpResponse.json(
          { error: "current password incorrect", code: "ERR_BAD_CREDENTIALS" },
          { status: 401 },
        ),
      ),
    );
    harness();
    await waitFor(() => screen.getByLabelText("Current password"));
    fireEvent.change(screen.getByLabelText("Current password"), { target: { value: "wrong" } });
    fireEvent.change(screen.getByLabelText("New password"), { target: { value: "newpw1" } });
    fireEvent.change(screen.getByLabelText("Confirm new password"), { target: { value: "newpw1" } });
    fireEvent.click(screen.getByRole("button", { name: /change password/i }));
    await waitFor(() =>
      expect(screen.getByText(/current password incorrect/i)).toBeInTheDocument(),
    );
  });
});
