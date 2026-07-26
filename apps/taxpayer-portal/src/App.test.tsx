// @vitest-environment jsdom
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import axe from "axe-core";
import { afterEach, describe, expect, it } from "vitest";
import { MemoryRouter } from "react-router";
import { App } from "./App";

afterEach(cleanup);

describe("portal accessibility baseline", () => {
  it("has no detectable structural accessibility violations", async () => {
    const { container } = render(
      <MemoryRouter>
        <App />
      </MemoryRouter>,
    );
    const results = await axe.run(container, {
      rules: { "color-contrast": { enabled: false } },
    });
    expect(results.violations).toEqual([]);
  });

  it("supports keyboard navigation and moves focus after routing", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <App />
      </MemoryRouter>,
    );
    const link = screen.getByRole("link", { name: "Payments" });
    link.focus();
    await user.keyboard("{Enter}");
    await waitFor(() =>
      expect(document.activeElement?.textContent).toBe("Payments"),
    );
  });

  it("announces form errors and associates them with the field", () => {
    render(
      <MemoryRouter initialEntries={["/components"]}>
        <App />
      </MemoryRouter>,
    );
    fireEvent.click(screen.getByRole("button", { name: "Validate example" }));
    const input = screen.getByLabelText("Payment reference");
    const alert = screen.getByRole("alert");
    expect(alert.textContent).toContain("Enter a payment reference");
    expect(input.getAttribute("aria-invalid")).toBe("true");
    expect(input.getAttribute("aria-describedby")).toContain(alert.id);
  });

  it("does not expose protected content after authorization failure", () => {
    render(
      <MemoryRouter initialEntries={["/restricted"]}>
        <App />
      </MemoryRouter>,
    );
    expect(screen.queryByText("Protected content")).toBeNull();
    expect(screen.getByRole("alert").textContent).toContain(
      "Access unavailable",
    );
  });

  it("presents an anonymous session without protected navigation", () => {
    render(
      <MemoryRouter>
        <App session={{ status: "anonymous" }} />
      </MemoryRouter>,
    );
    expect(
      screen.getByRole("heading", { name: "Sign in required" }),
    ).toBeTruthy();
    expect(screen.queryByRole("navigation")).toBeNull();
  });
});

describe("design-token contrast", () => {
  const luminance = (hex: string) => {
    const channels = hex
      .slice(1)
      .match(/.{2}/g)!
      .map((value) => Number.parseInt(value, 16) / 255)
      .map((value) =>
        value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4,
      );
    return 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
  };
  const ratio = (foreground: string, background: string) => {
    const values = [luminance(foreground), luminance(background)].sort(
      (a, b) => b - a,
    );
    return (values[0] + 0.05) / (values[1] + 0.05);
  };

  it("meets WCAG AA for primary text and controls", () => {
    expect(ratio("#17202a", "#ffffff")).toBeGreaterThanOrEqual(4.5);
    expect(ratio("#ffffff", "#075985")).toBeGreaterThanOrEqual(4.5);
    expect(ratio("#a61b1b", "#ffffff")).toBeGreaterThanOrEqual(4.5);
  });
});
