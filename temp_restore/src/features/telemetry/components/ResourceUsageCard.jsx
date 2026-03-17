import { clampPct, formatPct, formatBytes } from "../format";
import { Gauge } from "./Gauge";
import { Area, AreaChart, ResponsiveContainer } from "recharts";

function sev(p) {
  const v = clampPct(p);
  if (v == null) return "nodata";
  if (v < 70) return "ok";
  if (v < 90) return "warn";
  return "crit";
}

function barColor(p) {
  const s = sev(p);
  if (s === "ok") return "bg-emerald-500";
  if (s === "warn") return "bg-amber-500";
  if (s === "crit") return "bg-rose-500";
  return "bg-slate-600";
}

function statusText(p) {
  const s = sev(p);
  if (s === "nodata") return { t: "SEM DADO", cls: "text-slate-400" };
  if (s === "ok") return { t: "OK", cls: "text-emerald-300" };
  if (s === "warn") return { t: "ALERTA", cls: "text-amber-300" };
  return { t: "CRÍTICO", cls: "text-rose-300" };
}

function sparkColors(p) {
  const s = sev(p);
  if (s === "ok") return { stroke: "#34d399", fill: "rgba(52, 211, 153, 0.20)" };
  if (s === "warn") return { stroke: "#fbbf24", fill: "rgba(251, 191, 36, 0.22)" };
  if (s === "crit") return { stroke: "#fb7185", fill: "rgba(251, 113, 133, 0.22)" };
  return { stroke: "rgba(148,163,184,0.7)", fill: "rgba(148,163,184,0.12)" };
}

function toSparkData(series) {
  const arr = Array.isArray(series) ? series : [];
  return arr
    .slice()
    .sort((a, b) => (a.ts ?? 0) - (b.ts ?? 0))
    .map((p) => ({ ts: p.ts, v: Number.isFinite(Number(p.v)) ? Number(p.v) : 0 }));
}

export function ResourceUsageCard({ data }) {
  const memUsed = data?.memUsedBytes;
  const memTotal = data?.memTotalBytes;

  const cpu = data?.cpuPct;
  const mem = data?.memPct;
  const disk = data?.diskPct;

  // ✅ série real da memória (injetada pelo hook)
  const memSeries = toSparkData(data?.memSeries);
  const memLast = memSeries.length ? memSeries[memSeries.length - 1].v : undefined;
  const memCurrent = (typeof mem === "number" && Number.isFinite(mem)) ? mem : (typeof memLast === "number" && Number.isFinite(memLast) ? memLast : undefined);
  const sparkKey = memSeries.length ? memSeries[memSeries.length - 1].ts : "empty";

  const memStatus = statusText(memCurrent);
  const diskStatus = statusText(disk);
  const memSpark = sparkColors(memCurrent);

  return (
    <div className="rounded-2xl bg-slate-900/60 p-4 shadow-lg ring-1 ring-white/10">
      <div className="mb-3 flex items-center justify-between">
        <h3 className="text-sm font-semibold tracking-wide text-slate-100">[1] PERCENTUAIS</h3>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        {/* CPU */}
        <div className="rounded-xl bg-slate-950/40 p-3 ring-1 ring-white/5">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-xs text-slate-300">CPU (%)</span>
            <span className="text-xs text-slate-400">{cpu == null ? "—" : "Atual"}</span>
          </div>
          <div className="flex justify-center">
            <Gauge value={cpu} label="CPU" />
          </div>
        </div>

        {/* MEMÓRIA com sparkline */}
        <div className="rounded-xl bg-slate-950/40 p-3 ring-1 ring-white/5">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-xs text-slate-300">Memória (%)</span>
            <span className={`text-xs font-semibold ${memStatus.cls}`}>{memStatus.t}</span>
          </div>

          {/* barra */}
          <div className="h-3 w-full rounded-full bg-slate-800">
            <div
              className={`h-3 rounded-full ${barColor(memCurrent)}`}
              style={{ width: `${clampPct(mem) ?? 0}%` }}
            />
          </div>

          <div className="mt-2 flex items-end justify-between">
            <div className="flex flex-col">
              <div className="text-sm font-semibold text-slate-100">{formatPct(memCurrent)}</div>
              {memUsed != null && memTotal != null && (
                <div className="text-[10px] leading-tight text-slate-500">
                  {formatBytes(memUsed)} / {formatBytes(memTotal)}
                </div>
              )}
            </div>
            <div className="text-[11px] text-slate-500">últimos pontos</div>
          </div>

          {/* ✅ sparkline animado (cor muda pela criticidade atual) */}
          <div className="mt-2 h-10 w-full">
            {memSeries.length ? (
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={memSeries} margin={{ top: 2, right: 2, left: 2, bottom: 2 }}>
                  <defs>
                    <linearGradient id="memFill" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor={memSpark.stroke} stopOpacity={0.35} />
                      <stop offset="95%" stopColor={memSpark.stroke} stopOpacity={0.02} />
                    </linearGradient>
                  </defs>
                  <Area
                    key={sparkKey}
                    type="monotone"
                    dataKey="v"
                    stroke={memSpark.stroke}
                    fill="url(#memFill)"
                    fillOpacity={1}
                    strokeWidth={2}
                    isAnimationActive={true}
                    animationDuration={650}
                    animationEasing="ease-out"
                    dot={false}
                  />
                </AreaChart>
              </ResponsiveContainer>
            ) : (
              <div className="h-full rounded-lg bg-slate-900/30 ring-1 ring-white/5 flex items-center justify-center text-[11px] text-slate-500">
                sem série de memória
              </div>
            )}
          </div>
        </div>

        {/* DISCO */}
        <div className="rounded-xl bg-slate-950/40 p-3 ring-1 ring-white/5">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-xs text-slate-300">
              Disco (%) {data?.diskMount ? `(${data.diskMount})` : ""}
            </span>
            <span className={`text-xs font-semibold ${diskStatus.cls}`}>{diskStatus.t}</span>
          </div>
          <div className="flex justify-center">
            <Gauge value={disk} label="Disco" />
          </div>
        </div>
      </div>
    </div>
  );
}