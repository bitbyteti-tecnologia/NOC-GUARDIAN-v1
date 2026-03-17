import { AlertItem, TelemetryFlags } from "../models";
import { formatAgo } from "../format";
import { Led } from "./Led";

function badge(sev: AlertItem["severity"]) {
  if (sev === "critical") return "bg-rose-500/20 text-rose-200 ring-1 ring-rose-500/30";
  if (sev === "warning") return "bg-amber-500/20 text-amber-200 ring-1 ring-amber-500/30";
  return "bg-sky-500/20 text-sky-200 ring-1 ring-sky-500/30";
}

export function TelemetryStatusCard({ flags, alerts }: { flags?: TelemetryFlags; alerts?: AlertItem[] }) {
  const recent = (alerts ?? []).slice().sort((a, b) => b.ts - a.ts).slice(0, 5);

  return (
    <div className="rounded-2xl bg-slate-900/60 p-4 shadow-lg ring-1 ring-white/10">
      <div className="mb-3 flex items-center justify-between">
        <h3 className="text-sm font-semibold tracking-wide text-slate-100">[4] FLAGS TELEMETRIA</h3>
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        <div className="rounded-xl bg-slate-950/40 p-3 ring-1 ring-white/5">
          <Led ok={flags?.netOk} label="Telemetria NET" />
          <div className="mt-2" />
          <Led ok={flags?.diskOk} label="Telemetria DISK" />
        </div>

        <div className="rounded-xl bg-slate-950/40 p-3 ring-1 ring-white/5">
          <div className="mb-2 text-xs font-semibold text-slate-200">Histórico Alertas (1h)</div>
          {recent.length === 0 ? (
            <div className="text-sm text-slate-400">Sem alertas</div>
          ) : (
            <ul className="space-y-2">
              {recent.map((a) => (
                <li key={a.ts + a.message} className="flex items-start justify-between gap-2">
                  <div className="text-sm text-slate-200">{a.message}</div>
                  <div className="flex items-center gap-2">
                    <span className={`rounded-full px-2 py-0.5 text-[11px] ${badge(a.severity)}`}>{a.severity}</span>
                    <span className="text-[11px] text-slate-400">{formatAgo(a.ts)}</span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}
