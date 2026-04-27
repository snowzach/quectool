import { describe, it, expect } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "../test/server";
import { SMS } from "./SMS";

describe("SMS page", () => {
  it("lists messages, sends, and deletes", async () => {
    render(<SMS />);
    await waitFor(() => expect(screen.getByText("hi")).toBeInTheDocument());

    let sent: any = null;
    server.use(http.post("/api/modem/sms", async ({ request }) => {
      sent = await request.json();
      return HttpResponse.json({ index: 2 });
    }));
    fireEvent.change(screen.getByLabelText("To"), { target: { value: "+15555550101" } });
    fireEvent.change(screen.getByLabelText("Body"), { target: { value: "yo" } });
    fireEvent.click(screen.getByRole("button", { name: /send/i }));
    await waitFor(() => expect(sent).toEqual({ to: "+15555550101", body: "yo" }));

    let deleted = false;
    server.use(http.delete("/api/modem/sms/1", () => { deleted = true; return new HttpResponse(null, { status: 204 }); }));
    // Match the per-row "Delete" button exactly — header has "Delete read"/"Delete all".
    fireEvent.click(screen.getAllByRole("button", { name: /^Delete$/ })[0]);
    await waitFor(() => expect(deleted).toBe(true));
  });
});
