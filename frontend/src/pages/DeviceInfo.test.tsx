import { describe, it, expect } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { DeviceInfo } from "./DeviceInfo";

describe("DeviceInfo page", () => {
  it("renders modem info from /api/modem/info", async () => {
    render(<DeviceInfo />);
    await waitFor(() => expect(screen.getByText("RM520N-GL")).toBeInTheDocument());
    expect(screen.getByText("Quectel")).toBeInTheDocument();
    expect(screen.getByText(/sms/)).toBeInTheDocument();
  });
});
