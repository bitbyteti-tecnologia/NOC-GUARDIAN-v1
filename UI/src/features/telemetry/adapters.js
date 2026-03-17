function getByPath(obj, path) {
  if (!obj || !path) return undefined;
  const parts = path.split(".").filter(Boolean);
  let cur = obj;
  for (const p of parts) {
    if (cur == null) return undefined;
    cur = cur[p];
  }
  return cur;
}
function toNumber(v) {
  const n = typeof v === "string" ? Number(v) : v;
  return Number.isFinite(n) ? n : undefined;
}
function toBool(v) {
  if (typeof v === "boolean") return v;
  if (typeof v === "string") {
    const s = v.trim().toLowerCase();
    if (["true", "ok", "1", "yes", "ativo"].includes(s)) return true;
    if (["false", "nok", "0", "no", "inativo"].includes(s)) return false;
  }
  if (typeof v === "number") return v !== 0;
  return undefined;
}
function toTs(v) {
  const n = toNumber(v);
  if (n && n > 10_000_000_000) return n; // ms
  if (n) return n * 1000; // seconds
  const d = typeof v === "string" ? Date.parse(v) : NaN;
  return Number.isFinite(d) ? d : undefined;
}
export function mapTelemetry(host, rx, tx, read, write, netOk, diskOk, memSeries) {
  const lastMetrics = host.metrics || {};

  return {
    host: {
      ...host,
      uptime_sec: lastMetrics.uptime_sec,
      proc_count: lastMetrics.proc_count,
      thread_count: lastMetrics.thread_count,
      kthread_count: lastMetrics.kthread_count,
      running_procs: lastMetrics.running_procs,
      load_avg_1: lastMetrics.load_avg_1,
      load_avg_5: lastMetrics.load_avg_5,
      load_avg_15: lastMetrics.load_avg_15,
    },
    resources: {
      cpuPct: lastMetrics.cpu_percent,
      memPct: lastMetrics.mem_used_pct,
      diskPct: lastMetrics.disk_used_pct,
      memUsedBytes: host.memUsedBytes,
      memTotalBytes: host.memTotalBytes,
      memSeries,
    },
    network: {
      current: { rx: lastMetrics.net_rx_bps, tx: lastMetrics.net_tx_bps },
      series: { rx, tx, ok: netOk },
    },
    diskIO: {
      current: { read: lastMetrics.disk_read_bps, write: lastMetrics.disk_write_bps },
      series: { read, write, ok: diskOk },
    },
  };
}