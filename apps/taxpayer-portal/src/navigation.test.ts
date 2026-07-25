import { describe, expect, it } from "vitest";

import { taxpayerNavigationAreas } from "./navigation";

describe("taxpayer navigation", () => {
  it("provides unique labels for every initial portal area", () => {
    expect(taxpayerNavigationAreas.length).toBeGreaterThan(0);
    expect(new Set(taxpayerNavigationAreas).size).toBe(
      taxpayerNavigationAreas.length,
    );
  });
});
