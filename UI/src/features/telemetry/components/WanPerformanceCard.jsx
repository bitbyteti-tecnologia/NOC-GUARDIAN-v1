import { Area, CartesianGrid, ComposedChart, Legend, Line, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

function toChart(series) {
  if (!series?.length) return [];
  return series
    .slice()
    .sort((a, b) => a.ts - b.ts)
    .map((p) => ({ ts: p.ts, latencyMs: p.latencyMs ?? 0, lossPct: p.lossPct ?? 0 }));
}

function tickDate(ts) {
  const d = new Date(ts);
  return d.toLocaleDateString([], { month: "2-digit", day: "2-digit" });
}

function formatMs(v) {
  if (typeof v !== "number" || !Number.isFinite(v)) return "—";
  return `${v.toFixed(0)} ms`;
}

function formatPct(v) {
  if (typeof v !== "number" || !Number.isFinite(v)) return "—";
  return `${v.toFixed(2)}%`;
}

export function WanPerformanceCard({ series }) {
  const data = toChart(series);

  return (
    <div className="rounded-2xl bg-slate-900/60 p-4 shadow-lg ring-1 ring-white/10">
      <div className="mb-3 flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold tracking-wide text-slate-100">
            Desempenho da WAN - Últimos 30 Dias
          </h3>
          <div className="text-xs text-slate-400">
            Latência média e perda de pacotes na conexão externa.
          </div>
        </div>
      </div>
      <div className="h-56 w-full rounded-xl bg-slate-950/40 p-2 ring-1 ring-white/5">
        {data.length === 0 ? (
          <div className="flex h-full items-center justify-center text-sm text-slate-400">
            Sem série de WAN
          </div>
        ) : (
          <ResponsiveContainer width="100%" height="100%">
            <ComposedChart data={data} margin={{ top: 10, right: 20, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id="wanLossFill" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.35} />
                  <stop offset="95%" stopColor="#f59e0b" stopOpacity={0.05} />
                </linearGradient>
              </defs>
              <CartesianGrid stroke="rgba(148,163,184,0.18)" strokeDasharray="3 3" />
              <XAxis
                dataKey="ts"
                tickFormatter={tickDate}
                tick={{ fill: "rgba(226,232,240,0.7)", fontSize: 11 }}
                axisLine={{ stroke: "rgba(148,163,184,0.25)" }}
              />
              <YAxis
                yAxisId="latency"
                tick={{ fill: "rgba(226,232,240,0.7)", fontSize: 11 }}
                axisLine={{ stroke: "rgba(148,163,184,0.25)" }}
                label={{ value: "Latência (ms)", angle: -90, position: "insideLeft", fill: "rgba(226,232,240,0.6)" }}
              />
              <YAxis
                yAxisId="loss"
                orientation="right"
                tick={{ fill: "rgba(226,232,240,0.7)", fontSize: 11 }}
                axisLine={{ stroke: "rgba(148,163,184,0.25)" }}
                label={{ value: "Perda (%)", angle: 90, position: "insideRight", fill: "rgba(226,232,240,0.6)" }}
              />
              <Tooltip
                contentStyle={{ background: "rgba(2,6,23,0.95)", border: "1px solid rgba(255,255,255,0.1)" }}
                labelFormatter={(v) => tickDate(Number(v))}
                formatter={(v, name) => [
                  name === "latencyMs" ? formatMs(Number(v)) : formatPct(Number(v)),
                  name === "latencyMs" ? "Latência (ms)" : "Perda (%)",
                ]}
              />
              <Legend
                formatter={(v) => (v === "latencyMs" ? "Latência (ms)" : "Perda (%)")}
                wrapperStyle={{ color: "rgba(226,232,240,0.75)", fontSize: 12 }}
              />
              <Line yAxisId="latency" type="monotone" dataKey="latencyMs" stroke="#60a5fa" strokeWidth={2} dot={false} />
              <Area yAxisId="loss" type="monotone" dataKey="lossPct" stroke="#f59e0b" fill="url(#wanLossFill)" />
            </ComposedChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  );
}
