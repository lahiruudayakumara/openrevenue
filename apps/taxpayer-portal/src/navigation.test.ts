import { describe, expect, it } from "vitest";

import { taxpayerNavigation, visibleNavigation } from "./navigation";

describe("taxpayer navigation", () => {
  it("provides unique labels for every initial portal area", () => {
    expect(taxpayerNavigation.length).toBeGreaterThan(0);
    expect(new Set(taxpayerNavigation.map((item) => item.label)).size).toBe(
      taxpayerNavigation.length,
    );
  });

  it("hides navigation the session cannot authorize", () => {
    expect(
      visibleNavigation([]).every((item) => !item.requiredPermission),
    ).toBe(true);
  });
});
