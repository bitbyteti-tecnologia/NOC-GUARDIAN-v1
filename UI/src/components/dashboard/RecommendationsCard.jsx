import React from "react";

function SkeletonRow({ i }) {
  return (
    <div key={i} className="animate-pulse flex items-center gap-3">
      <div className="h-4 w-4 bg-slate-800/80 rounded" />
      <div className="h-4 w-56 bg-slate-800/80 rounded" />
    </div>
  );
}

export default function RecommendationsCard({ items = [], loading, error }) {
  return (
    <div className="rounded-2xl border border-slate-800 bg-slate-950/50 p-5 shadow-lg">
      <div className="flex items-center justify-between">
        <div className="text-sm font-semibold text-slate-100">Recomendações</div>
        <div className="text-xs text-slate-400">Ações sugeridas</div>
      </div>

      <div className="mt-4 space-y-3">
        {loading ? (
          [0, 1, 2].map((i) => <SkeletonRow key={i} i={i} />)
        ) : error ? (
          <div className="text-sm text-rose-200">Falha ao carregar recomendações.</div>
        ) : items.length === 0 ? (
          <div className="text-sm text-slate-400">Nenhuma recomendação no momento.</div>
        ) : (
          items.map((rec, idx) => (
            <div key={`${rec.type}-${idx}`} className="flex items-start gap-3">
              <div className="text-emerald-300">🛠</div>
              <div className="text-sm text-slate-100">{rec.message}</div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
