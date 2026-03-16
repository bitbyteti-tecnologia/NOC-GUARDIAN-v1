import { TelemetryVM } from "./models";
import { TelemetryFieldMap } from "./fieldMap";

function getByPath(obj: any, path?: string): any {
  if (!obj || !path) return undefined;
  const parts = path.split(".").filter(Boolean);
  let cur = obj;
  for (const p of parts) {
    if (cur == null) return undefined;
    cur = cur[p];
  }
  return cur;
}

function toNumber(v: any): number | undefined {
  const n = typeof v === "string" ? Number(v) : v;
  return Number.isFinite(n) ? n : undefined;
}

function toBool(v: any): boolean | undefined {
  if (typeof v === "boolean") return v;
  if (typeof v === "string") {
    const s = v.trim().toLowerCase();
    if (["true", "ok", "1", "yes", "ativo"].includes(s)) return true;
    if (["false", "nok", "0", "no", "inativo"].includes(s)) return false;
  }
  if (typeof v === "number") return v !== 0;
  return undefined;
}

function toTs(v: any): number | undefined {
  const n = toNumber(v);
  if (n && n > 10_000_000_000) return n; // já em ms
  if (n) return n * 1000; // provável seconds
  const d = typeof v === "string" ? Date.parse(v) : NaN;
  return Number.isFinite(d) ? d : undefined;
}

export function mapTelemetry(raw: any, map: TelemetryFieldMap): TelemetryVM {
  const host = {
    name: String(getByPath(raw, map.hostName) ?? ""),
    ip: getByPath(raw, map.hostIp),
    os: getByPath(raw, map.hostOs),
    uptime: getByPath(raw, map.hostUptime),
  };

  const resources = {
    cpuPct: toNumber(getByPath(raw, map.cpuPct)),
    memPct: toNumber(getByPath(raw, map.memPct)),
    diskPct: toNumber(getByPath(raw, map.diskPct)),
    diskMount: getByPath(raw, map.diskMount),
  };

  const rx = toNumber(getByPath(raw, map.netCurrentRxBps));
  const tx = toNumber(getByPath(raw, map.netCurrentTxBps));
  const networkSeriesRaw = getByPath(raw, map.netSeries);
  const networkSeries = Array.isArray(networkSeriesRaw)
    ? networkSeriesRaw
        .map((it: any) => ({
          ts: toTs(getByPath(it, map.netSeriesTs)) ?? 0,
          rxBps: toNumber(getByPath(it, map.netSeriesRxBps)),
          txBps: toNumber(getByPath(it, map.netSeriesTxBps)),
        }))
        .filter((p: any) => p.ts > 0)
    : undefined;

  const read = toNumber(getByPath(raw, map.diskReadBps));
  const write = toNumber(getByPath(raw, map.diskWriteBps));
  const diskSeriesRaw = getByPath(raw, map.diskSeries);
  const diskSeries = Array.isArray(diskSeriesRaw)
    ? diskSeriesRaw
        .map((it: any) => ({
          ts: toTs(getByPath(it, map.diskSeriesTs)) ?? 0,
          readBps: toNumber(getByPath(it, map.diskSeriesReadBps)),
          writeBps: toNumber(getByPath(it, map.diskSeriesWriteBps)),
        }))
        .filter((p: any) => p.ts > 0)
    : undefined;

  const netOk = toBool(getByPath(raw, map.flagNetOk));
  const diskOk = toBool(getByPath(raw, map.flagDiskOk));

  const alertsRaw = getByPath(raw, map.alerts);
  const alerts = Array.isArray(alertsRaw)
    ? alertsRaw
        .map((it: any) => {
          const severityRaw = getByPath(it, map.alertSeverity);
          const severity =
            severityRaw === "critical" || severityRaw === "warning" || severityRaw === "info"
              ? severityRaw
              : "info";
          return {
            ts: toTs(getByPath(it, map.alertTs)) ?? Date.now(),
            message: String(getByPath(it, map.alertMessage) ?? ""),
            severity,
          };
        })
        .filter((a: any) => a.message)
    : undefined;

  return {
    host,
    resources,
    network: {
      current: { rxBps: rx, txBps: tx, totalBps: (rx ?? 0) + (tx ?? 0) },
      series: networkSeries,
    },
    diskIO: {
      current: { readBps: read, writeBps: write },
      series: diskSeries,
    },
    flags: { netOk, diskOk },
    alerts,
  };
}
