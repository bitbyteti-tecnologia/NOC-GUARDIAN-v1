import { useEffect, useMemo, useState } from "react";
import { buildTelemetryVMFromHost } from "./fromHost";

function normalizePoints(payload) {
  const arr = Array.isArray(payload) ? payload : (payload?.points || []);
  return Array.isArray(arr) ? arr : [];
}

function toTs(p) {
  const t = p?.ts ?? p?.t ?? p?.time ?? p?.x;
  if (typeof t === "number") return (t > 10_000_000_000 ? t : t * 1000);
  const parsed = Date.parse(String(t));
  return Number.isFinite(parsed) ? parsed : 0;
}

function toVal(p) {
  const v = p?.v ?? p?.value ?? p?.y;
  const n = typeof v === "string" ? Number(v) : v;
  return Number.isFinite(n) ? n : 0;
}

export function useTelemetryFromApi({
  api,
  tenantId,
  host,
  window = "1h",
  enabled = true,
  pollMs = 15000, // ✅ NOVO: intervalo de atualização real (ms)
}) {
  const [rx, setRx] = useState([]);
  const [tx, setTx] = useState([]);
  const [read, setRead] = useState([]);
  const [write, setWrite] = useState([]);
  const [memSeries, setMemSeries] = useState([]); // ✅ série memória
  const [wanLatency, setWanLatency] = useState([]);
  const [wanLoss, setWanLoss] = useState([]);

  const hostname = host?.hostname;

  useEffect(() => {
    let alive = true;
    let timer = null;

    async function load() {
      if (!enabled || !api || !tenantId || !hostname) return;

      const base =
        `/api/v1/tenants/${tenantId}/dashboard/series` +
        `?hostname=${encodeURIComponent(hostname)}` +
        `&window=${encodeURIComponent(window)}`;

      const METRICS = {
        rx: "net_rx_bps",
        tx: "net_tx_bps",
        read: "disk_read_bps",
        write: "disk_write_bps",
        mem: "mem_used_pct",
        memUsed: "mem_used_bytes",
        memTotal: "mem_total_bytes",
        wanLatency: "wan_latency_ms",
        wanLoss: "wan_packet_loss_pct",
      };

      try {
        const [a, b, c, d, g, h, i, j, k] = await Promise.all([
          api.get(`${base}&metric=${encodeURIComponent(METRICS.rx)}`),
          api.get(`${base}&metric=${encodeURIComponent(METRICS.tx)}`),
          api.get(`${base}&metric=${encodeURIComponent(METRICS.read)}`),
          api.get(`${base}&metric=${encodeURIComponent(METRICS.write)}`),
          api.get(`${base}&metric=${encodeURIComponent(METRICS.mem)}`),
          api.get(`${base}&metric=${encodeURIComponent(METRICS.memUsed)}`),
          api.get(`${base}&metric=${encodeURIComponent(METRICS.memTotal)}`),
          api.get(`${base}&metric=${encodeURIComponent(METRICS.wanLatency)}`),
          api.get(`${base}&metric=${encodeURIComponent(METRICS.wanLoss)}`),
        ]);

        if (!alive) return;

        setRx(normalizePoints(a.data).map((p) => ({ ts: toTs(p), v: toVal(p) })).filter((p) => p.ts));
        setTx(normalizePoints(b.data).map((p) => ({ ts: toTs(p), v: toVal(p) })).filter((p) => p.ts));
        setRead(normalizePoints(c.data).map((p) => ({ ts: toTs(p), v: toVal(p) })).filter((p) => p.ts));
        setWrite(normalizePoints(d.data).map((p) => ({ ts: toTs(p), v: toVal(p) })).filter((p) => p.ts));
        setMemSeries(normalizePoints(g.data).map((p) => ({ ts: toTs(p), v: toVal(p) })).filter((p) => p.ts));
        setWanLatency(normalizePoints(j.data).map((p) => ({ ts: toTs(p), v: toVal(p) })).filter((p) => p.ts));
        setWanLoss(normalizePoints(k.data).map((p) => ({ ts: toTs(p), v: toVal(p) })).filter((p) => p.ts));

        const lastUsed = normalizePoints(h.data).pop()?.v;
        const lastTotal = normalizePoints(i.data).pop()?.v;
        if (lastUsed !== undefined) host.memUsedBytes = lastUsed;
        if (lastTotal !== undefined) host.memTotalBytes = lastTotal;
      } catch {
        if (!alive) return;
        setRx([]); setTx([]); setRead([]); setWrite([]);
        setMemSeries([]);
        setWanLatency([]); setWanLoss([]);
      }
    }

    // carrega já
    load();

    // ✅ polling enquanto enabled
    if (enabled && pollMs > 0) {
      timer = setInterval(load, pollMs);
    }

    return () => {
      alive = false;
      if (timer) clearInterval(timer);
    };
  }, [api, tenantId, hostname, window, enabled, pollMs]);

  const vm = useMemo(() => {
    const base = buildTelemetryVMFromHost(host);

    // injeta série de memória pro card
    base.resources = {
      ...(base.resources || {}),
      memSeries,
    };

    // Rede
    const rxLast = rx.length ? rx[rx.length - 1].v : undefined;
    const txLast = tx.length ? tx[tx.length - 1].v : undefined;

    base.network = {
      current: (rxLast == null && txLast == null)
        ? undefined
        : { rxBps: rxLast, txBps: txLast, totalBps: (rxLast ?? 0) + (txLast ?? 0) },
      series: (rx.length || tx.length)
        ? rx.map((p, i) => ({ ts: p.ts, rxBps: p.v, txBps: tx[i]?.v ?? 0 }))
        : undefined,
    };

    // Disco I/O
    const readLast = read.length ? read[read.length - 1].v : undefined;
    const writeLast = write.length ? write[write.length - 1].v : undefined;

    base.diskIO = {
      current: (readLast == null && writeLast == null)
        ? undefined
        : { readBps: readLast, writeBps: writeLast },
      series: (read.length || write.length)
        ? read.map((p, i) => ({ ts: p.ts, readBps: p.v, writeBps: write[i]?.v ?? 0 }))
        : undefined,
    };

    // LAN (usa série de rede atual)
    base.lan = {
      series: (rx.length || tx.length)
        ? rx.map((p, i) => ({ ts: p.ts, rxBps: p.v, txBps: tx[i]?.v ?? 0 }))
        : undefined,
    };

    // WAN (latência e perda de pacotes)
    base.wan = {
      series: (wanLatency.length || wanLoss.length)
        ? wanLatency.map((p, i) => ({ ts: p.ts, latencyMs: p.v, lossPct: wanLoss[i]?.v ?? 0 }))
        : undefined,
    };

    // Flags
    const netOk = rx.length || tx.length ? true : undefined;
    const diskOk = read.length || write.length ? true : undefined;
    base.flags = { netOk, diskOk };

    return base;
  }, [host, rx, tx, read, write, memSeries, wanLatency, wanLoss]);

  return { vm };
}
