import { describe, expect, it } from "vitest";
import { hundredths, lb, money, qty, shiftDate, toCents, toHundredths } from "./format";

describe("format", () => {
  it("money", () => {
    expect(money(599)).toBe("$5.99");
    expect(money(0)).toBe("$0.00");
    expect(money(123456)).toBe("$1,234.56");
  });
  it("hundredths and lb", () => {
    expect(hundredths(250)).toBe("2.5");
    expect(hundredths(300)).toBe("3");
    expect(lb(11850)).toBe("118.50 lb");
    expect(lb(undefined)).toBe("—");
  });
  it("qty by unit", () => {
    expect(qty(200, "case")).toBe("2 cases");
    expect(qty(100, "case")).toBe("1 case");
    expect(qty(1000, "each")).toBe("10 each");
    expect(qty(250, "lb")).toBe("2.50 lb");
  });
  it("parsing", () => {
    expect(toHundredths("2.5")).toBe(250);
    expect(toHundredths("2")).toBe(200);
    expect(toHundredths("abc")).toBeNull();
    expect(toHundredths("-1")).toBeNull();
    expect(toCents("5.99")).toBe(599);
    expect(toCents("$5.99")).toBe(599);
  });
  it("shiftDate crosses month, year, and leap-year boundaries independent of local timezone", () => {
    expect(shiftDate("2026-03-08", 1)).toBe("2026-03-09");
    expect(shiftDate("2026-12-31", 1)).toBe("2027-01-01");
    expect(shiftDate("2026-03-01", -1)).toBe("2026-02-28");
  });
});
