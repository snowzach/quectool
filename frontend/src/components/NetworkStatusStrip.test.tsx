import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { NetworkStatusStrip } from "./NetworkStatusStrip";

const cell = {
  tech: "NR5G-SA", mcc: "310", mnc: "260", cell_id: "X",
  pci: 187, band: "n71", bandwidth: "100",
  signal: { rsrp: -78, rsrq: -10, sinr: 12, tech: "NR5G-SA" },
  scells: [{ tech: "LTE", mcc: "", mnc: "", cell_id: "", pci: 311,
    band: "B2", bandwidth: "20",
    signal: { rsrp: -90, rsrq: -12, sinr: 8, tech: "LTE" } }],
};

describe("NetworkStatusStrip", () => {
  it("renders serving cell summary, CA chips, and per-tech lock chips", () => {
    render(
      <NetworkStatusStrip
        cell={cell as any}
        cellLock={{ lock_4g: false, lock_5g: true }}
        onLockCurrent={async () => {}}
        onUnlock={async () => {}}
      />,
    );
    expect(screen.getByText(/n71/)).toBeInTheDocument();
    expect(screen.getByText(/PCI 187/)).toBeInTheDocument();
    expect(screen.getByText(/-78/)).toBeInTheDocument();
    expect(screen.getByText(/B2/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /unlock 5g/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /lock 4g to current/i })).toBeInTheDocument();
  });

  it("calls onUnlock when 5G unlock is clicked", () => {
    const onUnlock = vi.fn().mockResolvedValue(undefined);
    render(
      <NetworkStatusStrip
        cell={cell as any}
        cellLock={{ lock_4g: false, lock_5g: true }}
        onLockCurrent={async () => {}}
        onUnlock={onUnlock}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: /unlock 5g/i }));
    expect(onUnlock).toHaveBeenCalledWith("5g");
  });

  it("calls onLockCurrent when 4G lock is clicked", () => {
    const onLockCurrent = vi.fn().mockResolvedValue(undefined);
    render(
      <NetworkStatusStrip
        cell={cell as any}
        cellLock={{ lock_4g: false, lock_5g: false }}
        onLockCurrent={onLockCurrent}
        onUnlock={async () => {}}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: /lock 4g to current/i }));
    expect(onLockCurrent).toHaveBeenCalledWith("4g");
  });

  it("renders an acquiring placeholder when cell is null", () => {
    render(
      <NetworkStatusStrip
        cell={null}
        cellLock={{ lock_4g: false, lock_5g: false }}
        onLockCurrent={async () => {}}
        onUnlock={async () => {}}
      />,
    );
    expect(screen.getByText(/acquiring/i)).toBeInTheDocument();
  });
});
