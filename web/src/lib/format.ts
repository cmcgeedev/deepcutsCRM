const usd = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" });

export function money(cents: number): string {
  return usd.format(cents / 100);
}

export function hundredths(v: number): string {
  return (v / 100).toString();
}

export function lb(v: number | undefined | null): string {
  return v == null ? "—" : (v / 100).toFixed(2) + " lb";
}

export function qty(v: number, unit: string): string {
  if (unit === "lb") return (v / 100).toFixed(2) + " lb";
  const n = v / 100;
  if (unit === "case") return `${n} ${n === 1 ? "case" : "cases"}`;
  return `${n} ${unit}`;
}

function parseDecimal(s: string): number | null {
  const cleaned = s.replace(/[$,\s]/g, "");
  if (!/^\d+(\.\d{0,2})?$/.test(cleaned)) return null;
  return Math.round(parseFloat(cleaned) * 100);
}

export const toHundredths = parseDecimal;
export const toCents = parseDecimal;
