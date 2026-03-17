import { Bar, BarChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { formatBps } from "../format";
function toChart(series) {
  if (!series?.length) return [];
  return series.slice().sort((a, b) => a.ts - b.ts).map((p) => ({ ts: p.ts, readBps: p.readBps ?? 0, writeBps: p.writeBps ?? 0 }));
}
function tickTime(ts) {
  const d = new Date(ts);
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}
export function DiskIoCard({ current, series }) {
  const data = toChart(series);
  return (
    <div className="rounded-2xl bg-slate-900/60 p-4 shadow-lg ring-1 ring-white/10">
      <div className="mb-3 flex items-center justify-between">
        <h3 className="text-sm font-semibold tracking-wide text-slate-100">[3] DISCO I/O (bytes/s)</h3>
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <div className="rounded-xl bg-slate-950/40 p-3 ring-1 ring-white/5">
          <div className="text-xs text-slate-300">Disco Read</div>
          <div className="mt-1 text-lg font-semibold text-slate-100">{formatBps(current?.readBps)}</div>
        </div>
        <div className="rounded-xl bg-slate-950/40 p-3 ring-1 ring-white/5">
          <div className="text-xs text-slate-300">Disco Write</div>
          <div className="mt-1 text-lg font-semibold text-slate-100">{formatBps(current?.writeBps)}</div>
        </div>
      </div>
      <div className="mt-3 h-52 w-full rounded-xl bg-slate-950/40 p-2 ring-1 ring-white/5">
        {data.length === 0 ? (
          <div className="flex h-full items-center justify-center text-sm text-slate-400">Sem série de I/O</div>
        ) : (
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={data} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
              <CartesianGrid stroke="rgba(148,163,184,0.18)" strokeDasharray="3 3" />
              <XAxis dataKey="ts" tickFormatter={tickTime} tick={{ fill: "rgba(226,232,240,0.7)", fontSize: 11 }} axisLine={{ stroke: "rgba(148,163,184,0.25)" }} />
              <YAxis tick={{ fill: "rgba(226,232,240,0.7)", fontSize: 11 }} axisLine={{ stroke: "rgba(148,163,184,0.25)" }} />
              <Tooltip contentStyle={{ background: "rgba(2,6,23,0.95)", border: "1px solid rgba(255,255,255,0.1)" }} labelFormatter={(v) => tickTime(Number(v))} formatter={(v, name) => [formatBps(Number(v)), name === "readBps" ? "Read" : "Write"]} />
              <Legend formatter={(v) => (v === "readBps" ? "Read" : "Write")} wrapperStyle={{ color: "rgba(226,232,240,0.75)", fontSize: 12 }} />
              <Bar dataKey="readBps" fill="#a78bfa" radius={[6, 6, 0, 0]} />
              <Bar dataKey="writeBps" fill="#f59e0b" radius={[6, 6, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  );
}