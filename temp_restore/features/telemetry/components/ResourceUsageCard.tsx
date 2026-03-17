import { ResourceUsage } from "../models";
import { clampPct, formatPct } from "../format";
import { Gauge } from "./Gauge";

function barColor(p?: number) {
  const v = clampPct(p);
  if (v == null) return "bg-slate-600";
  if (v < 70) return "bg-emerald-500";
  if (v < 90) return "bg-amber-500";
  return "bg-rose-500";
}

function statusText(p?: number) {
  const v = clampPct(p);
  if (v == null) return { t: "SEM DADO", cls: "text-slate-400" };
  if (v < 70) return { t: "OK", cls: "text-emerald-300" };
  if (v < 90) return { t: "AVISO", cls: "text-amber-300" };
  return { t: "CRÍTICO", cls: "text-rose-300" };
}

export function ResourceUsageCard({ data }: { data: ResourceUsage }) {
  const cpu = data.cpuPct;
  const mem = data.memPct;
  const disk = data.diskPct;

  const memStatus = statusText(mem);
  const diskStatus = statusText(disk);

  return (
    <div className="rounded-2xl bg-slate-900/60 p-4 shadow-lg ring-1 ring-white/10">
      <div className="mb-3 flex items-center justify-between">
        <h3 className="text-sm font-semibold tracking-wide text-slate-100">[1] PERCENTUAIS</h3>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <div className="rounded-xl bg-slate-950/40 p-3 ring-1 ring-white/5">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-xs text-slate-300">CPU (%)</span>
            <span className="text-xs text-slate-400">{cpu == null ? "—" : "Atual"}</span>
          </div>
          <Gauge value={cpu} label="CPU" />
        </div>

        <div className="rounded-xl bg-slate-950/40 p-3 ring-1 ring-white/5">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-xs text-slate-300">Memória (%)</span>
            <span className={`text-xs font-semibold ${memStatus.cls}`}>{memStatus.t}</span>
          </div>
          <div className="h-3 w-full rounded-full bg-slate-800">
            <div
              className={`h-3 rounded-full ${barColor(mem)}`}
              style={{ width: `${clampPct(mem) ?? 0}%` }}
            />
          </div>
          <div className="mt-2 text-sm font-semibold text-slate-100">{formatPct(mem)}</div>
        </div>

        <div className="rounded-xl bg-slate-950/40 p-3 ring-1 ring-white/5">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-xs text-slate-300">
              Disco (%) {data.diskMount ? `(${data.diskMount})` : ""}
            </span>
            <span className={`text-xs font-semibold ${diskStatus.cls}`}>{diskStatus.t}</span>
          </div>
          <Gauge value={disk} label="Disco" />
        </div>
      </div>
    </div>
  );
}
