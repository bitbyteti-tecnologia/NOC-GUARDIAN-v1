import React from "react";

function iconFor(type) {
  if (type === "anomaly") return "⚠";
  if (type === "instability") return "◆";
  return "•";
}

function SkeletonRow({ i }) {
  return (
    <div key={i} className="animate-pulse space-y-2">
      <div className="h-4 w-52 bg-slate-800/80 rounded" />
      <div className="h-3 w-32 bg-slate-800/80 rounded" />
    </div>
  );
}

export default function InsightsCard({ items = [], loading, error }) {
  return (
    <div className="rounded-2xl border border-slate-800 bg-slate-950/50 p-5 shadow-lg">
      <div className="flex items-center justify-between">
        <div className="text-sm font-semibold text-slate-100">Insights</div>
        <div className="text-xs text-slate-400">Anomalias e padrões</div>
      </div>

      <div className="mt-4 space-y-3">
        {loading ? (
          [0, 1, 2].map((i) => <SkeletonRow key={i} i={i} />)
        ) : error ? (
          <div className="text-sm text-rose-200">Falha ao carregar insights.</div>
        ) : items.length === 0 ? (
          <div className="text-sm text-slate-400">Nenhum insight ativo.</div>
        ) : (
          items.map((ins, idx) => (
            <div key={`${ins.type}-${idx}`} className="flex items-start gap-3">
              <div className="mt-1 text-lg">{iconFor(ins.type)}</div>
              <div className="min-w-0">
                <div className="text-sm text-slate-100">{ins.message}</div>
                <div className="text-xs text-slate-400 mt-1 flex flex-wrap gap-2">
                  {ins.metric ? <span>Métrica: <b className="text-slate-200">{ins.metric}</b></span> : null}
                  {typeof ins.change_percent === "number" ? (
                    <span>Δ <b className="text-slate-200">{ins.change_percent.toFixed(1)}%</b></span>
                  ) : null}
                  {ins.context ? <span>{ins.context}</span> : null}
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
