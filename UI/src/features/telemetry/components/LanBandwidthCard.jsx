import { Area, AreaChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

function bpsToMbps(bps) {
  if (typeof bps !== "number" || !Number.isFinite(bps)) return undefined;
  return (bps * 8) / 1_000_000;
}

function toChart(series) {
  if (!series?.length) return [];
  return series
    .slice()
    .sort((a, b) => a.ts - b.ts)
    .map((p) => ({
      ts: p.ts,
      rxMbps: bpsToMbps(p.rxBps ?? p.rxBps),
      txMbps: bpsToMbps(p.txBps ?? p.txBps),
    }));
}

function tickTime(ts) {
  const d = new Date(ts);
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function formatMbps(v) {
  if (typeof v !== "number" || !Number.isFinite(v)) return "—";
  return `${v.toFixed(2)} Mbps`;
}

export function LanBandwidthCard({ series }) {
  const data = toChart(series);

  return (
    <div className="rounded-2xl bg-slate-900/60 p-4 shadow-lg ring-1 ring-white/10">
      <div className="mb-3 flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold tracking-wide text-slate-100">
            Utilização de Largura de Banda LAN - Últimas 24 Horas
          </h3>
          <div className="text-xs text-slate-400">
            Tráfego interno entre dispositivos na rede local.
          </div>
        </div>
      </div>
      <div className="h-56 w-full rounded-xl bg-slate-950/40 p-2 ring-1 ring-white/5">
        {data.length === 0 ? (
          <div className="flex h-full items-center justify-center text-sm text-slate-400">
            Sem série de LAN
          </div>
        ) : (
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={data} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id="lanRxFill" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#38bdf8" stopOpacity={0.35} />
                  <stop offset="95%" stopColor="#38bdf8" stopOpacity={0.05} />
                </linearGradient>
                <linearGradient id="lanTxFill" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#22c55e" stopOpacity={0.35} />
                  <stop offset="95%" stopColor="#22c55e" stopOpacity={0.05} />
                </linearGradient>
              </defs>
              <CartesianGrid stroke="rgba(148,163,184,0.18)" strokeDasharray="3 3" />
              <XAxis
                dataKey="ts"
                tickFormatter={tickTime}
                tick={{ fill: "rgba(226,232,240,0.7)", fontSize: 11 }}
                axisLine={{ stroke: "rgba(148,163,184,0.25)" }}
              />
              <YAxis
                tick={{ fill: "rgba(226,232,240,0.7)", fontSize: 11 }}
                axisLine={{ stroke: "rgba(148,163,184,0.25)" }}
              />
              <Tooltip
                contentStyle={{ background: "rgba(2,6,23,0.95)", border: "1px solid rgba(255,255,255,0.1)" }}
                labelFormatter={(v) => tickTime(Number(v))}
                formatter={(v, name) => [formatMbps(Number(v)), name === "rxMbps" ? "Download (LAN)" : "Upload (LAN)"]}
              />
              <Legend
                formatter={(v) => (v === "rxMbps" ? "Download (LAN)" : "Upload (LAN)")}
                wrapperStyle={{ color: "rgba(226,232,240,0.75)", fontSize: 12 }}
              />
              <Area type="monotone" dataKey="rxMbps" stackId="lan" stroke="#38bdf8" fill="url(#lanRxFill)" strokeWidth={2} />
              <Area type="monotone" dataKey="txMbps" stackId="lan" stroke="#22c55e" fill="url(#lanTxFill)" strokeWidth={2} />
            </AreaChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  );
}
