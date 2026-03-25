import React from "react";

function sevBadge(sev) {
  if (sev === "critical") return "bg-rose-500/15 text-rose-200 border-rose-500/30";
  if (sev === "warning") return "bg-amber-500/15 text-amber-200 border-amber-500/30";
  return "bg-sky-500/15 text-sky-200 border-sky-500/30";
}

function SkeletonRow({ i }) {
  return (
    <div key={i} className="animate-pulse flex items-center justify-between gap-3">
      <div className="h-4 w-40 bg-slate-800/80 rounded" />
      <div className="h-4 w-14 bg-slate-800/80 rounded" />
    </div>
  );
}

export default function IncidentsCard({ items = [], loading, error }) {
  return (
    <div className="rounded-2xl border border-slate-800 bg-slate-950/50 p-5 shadow-lg">
      <div className="flex items-center justify-between">
        <div className="text-sm font-semibold text-slate-100">Top Incidentes</div>
        <div className="text-xs text-slate-400">Prioridade e impacto</div>
      </div>

      <div className="mt-4 space-y-3">
        {loading ? (
          [0, 1, 2].map((i) => <SkeletonRow key={i} i={i} />)
        ) : error ? (
          <div className="text-sm text-rose-200">Falha ao carregar incidentes.</div>
        ) : items.length === 0 ? (
          <div className="text-sm text-slate-400">Nenhum incidente ativo.</div>
        ) : (
          items.map((inc) => (
            <div key={inc.incident_id || inc.id} className="flex items-center justify-between gap-3">
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <span className={["px-2 py-0.5 rounded-full text-xs border", sevBadge(inc.severity)].join(" ")}
                  >
                    {String(inc.severity || "info").toUpperCase()}
                  </span>
                  <span className="text-sm font-semibold text-slate-100 truncate">
                    {inc.root_event || "incidente"}
                  </span>
                </div>
                <div className="text-xs text-slate-400 mt-1">
                  Impacto: <span className="text-slate-200 font-semibold">{inc.impact_count ?? 0}</span>
                </div>
              </div>
              <div className="text-right">
                <div className="text-xs text-slate-400">Prioridade</div>
                <div className="text-sm font-semibold text-slate-100">{inc.priority_score ?? 0}</div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
