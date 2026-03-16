export function clampPct(v?: number): number | undefined {
  if (typeof v !== "number" || !Number.isFinite(v)) return undefined;
  return Math.max(0, Math.min(100, v));
}

export function formatPct(v?: number): string {
  const c = clampPct(v);
  return typeof c === "number" ? `${c.toFixed(0)}%` : "—";
}

export function formatBps(bytesPerSec?: number): string {
  if (typeof bytesPerSec !== "number" || !Number.isFinite(bytesPerSec)) return "—";
  const units = ["B/s", "KB/s", "MB/s", "GB/s", "TB/s"];
  let v = bytesPerSec;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  const digits = i === 0 ? 0 : i === 1 ? 0 : 1;
  return `${v.toFixed(digits)} ${units[i]}`;
}

export function formatAgo(ts: number): string {
  const diff = Date.now() - ts;
  const min = Math.round(diff / 60000);
  if (min <= 1) return "agora";
  if (min < 60) return `${min}m atrás`;
  const h = Math.round(min / 60);
  return `${h}h atrás`;
}
