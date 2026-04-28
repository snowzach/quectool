import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "../test/server";
import { CellSurveyPanel } from "./CellSurveyPanel";

describe("CellSurveyPanel", () => {
  it("opens confirm modal when row Lock clicked, fires onLock on confirm", async () => {
    server.use(http.post("/api/modem/cell-survey", () =>
      HttpResponse.json([
        { tech: "NR5G", band: 71, freq: 132000, pci: 42, rsrp: -86,
          rsrq: -10, scs: 1, mcc: "310", mnc: "260", cell_id: "abc" },
      ])));
    const onLock = vi.fn().mockResolvedValue(undefined);
    render(
      <CellSurveyPanel currentCell={null} onLock={onLock} onError={() => {}} onSuccess={() => {}} />,
    );
    fireEvent.click(screen.getByRole("button", { name: /run survey/i }));
    await waitFor(() => screen.getByText("42"));
    fireEvent.click(screen.getByRole("button", { name: /^lock$/i }));
    const dialog = await screen.findByRole("dialog");
    fireEvent.click(within(dialog).getByRole("button", { name: /^lock$/i }));
    await waitFor(() => expect(onLock).toHaveBeenCalled());
  });

  it("does not call onLock when cancel clicked", async () => {
    server.use(http.post("/api/modem/cell-survey", () =>
      HttpResponse.json([
        { tech: "LTE", band: 12, freq: 5230, pci: 215, rsrp: -94,
          rsrq: -12, scs: 0, mcc: "310", mnc: "260", cell_id: "x" },
      ])));
    const onLock = vi.fn();
    render(
      <CellSurveyPanel currentCell={null} onLock={onLock} onError={() => {}} onSuccess={() => {}} />,
    );
    fireEvent.click(screen.getByRole("button", { name: /run survey/i }));
    await waitFor(() => screen.getByText("215"));
    fireEvent.click(screen.getByRole("button", { name: /^lock$/i }));
    const dialog = await screen.findByRole("dialog");
    fireEvent.click(within(dialog).getByRole("button", { name: /cancel/i }));
    expect(onLock).not.toHaveBeenCalled();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("highlights the current serving cell in results", async () => {
    server.use(http.post("/api/modem/cell-survey", () =>
      HttpResponse.json([
        { tech: "NR5G", band: 71, freq: 132000, pci: 187, rsrp: -78,
          rsrq: -10, scs: 1, mcc: "310", mnc: "260", cell_id: "current" },
        { tech: "NR5G", band: 41, freq: 510000, pci: 42, rsrp: -86,
          rsrq: -10, scs: 1, mcc: "310", mnc: "260", cell_id: "other" },
      ])));
    render(
      <CellSurveyPanel
        currentCell={{ pci: 187, band: "n71" } as any}
        onLock={async () => {}}
        onError={() => {}}
        onSuccess={() => {}}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: /run survey/i }));
    await waitFor(() => screen.getByText("187"));
    expect(screen.getByText("(current)")).toBeInTheDocument();
  });
});
