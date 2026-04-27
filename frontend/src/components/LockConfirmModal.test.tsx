import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { LockConfirmModal } from "./LockConfirmModal";

describe("LockConfirmModal", () => {
  it("renders title and body, calls onConfirm on confirm click", () => {
    const onConfirm = vi.fn();
    const onCancel = vi.fn();
    render(
      <LockConfirmModal
        title="Lock 5G"
        body="Lock to n71 PCI 187?"
        confirmLabel="Lock"
        onConfirm={onConfirm}
        onCancel={onCancel}
      />,
    );
    expect(screen.getByText("Lock 5G")).toBeInTheDocument();
    expect(screen.getByText(/n71 PCI 187/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Lock" }));
    expect(onConfirm).toHaveBeenCalledOnce();
    expect(onCancel).not.toHaveBeenCalled();
  });

  it("calls onCancel on cancel click", () => {
    const onCancel = vi.fn();
    render(
      <LockConfirmModal
        title="X"
        body="y"
        confirmLabel="OK"
        onConfirm={() => {}}
        onCancel={onCancel}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: /cancel/i }));
    expect(onCancel).toHaveBeenCalledOnce();
  });
});
