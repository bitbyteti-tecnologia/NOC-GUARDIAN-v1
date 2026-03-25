import React from "react";

function statusColor(status) {
  if (status === "healthy") return "from-emerald-500/20 to-emerald-500/5 border-emerald-500/30";
  if (status === "warning") return "from-amber-500/20 to-amber-500/5 border-amber-500/30";
  return "from-rose-500/20 to-rose-500/5 border-rose-500/30";
}

function statusLabel(status) {
  if (status === "healthy") return "Saudável";
  if (status === "warning") return "Atenção";
  return "Crítico";
}

function trendLabel(trend) {
  if (trend === "improving") return { icon: "↑", label: "Melhorando", cls: "text-emerald-300" };
  if (trend === "degrading") return { icon: "↓", label: "Piorando", cls: "text-rose-300" };
  return { icon: "→", label: "Estável", cls: "text-slate-300" };
}

function Skeleton() {
  return (
    <div className="animate-pulse space-y-3">
      <div className="h-5 w-32 bg-slate-800/80 rounded" />
      <div className="h-12 w-24 bg-slate-800/80 rounded" />
      <div className="h-4 w-full bg-slate-800/80 rounded" />
      <div className="h-4 w-2/3 bg-slate-800/80 rounded" />
    </div>
  );
}

export default function HealthCard({ data, loading, error }) {
  if (loading) {
    return (
      <div className="rounded-2xl border border-slate-800 bg-slate-950/50 p-5 shadow-lg">
        <Skeleton />
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-2xl border border-rose-500/30 bg-rose-950/20 p-5 shadow-lg">
        <div className="text-sm text-rose-200">Falha ao carregar inteligência.</div>
        <div className="text-xs text-rose-300/80 mt-1">Tente atualizar o dashboard.</div>
      </div>
    );
  }

  const score = data?.health_score ?? 0;
  const status = data?.status || "critical";
  const summary = data?.summary || "Sem resumo disponível.";
  const trend = trendLabel(data?.trend || "stable");

  return (
    <div className={[
      "rounded-2xl border bg-gradient-to-br p-5 shadow-lg",
      statusColor(status),
    ].join(" ")}
    >
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="text-xs uppercase tracking-widest text-slate-300">Health Score</div>
          <div className="text-4xl font-extrabold text-slate-100 mt-1">{score}</div>
          <div className="mt-2 inline-flex items-center gap-2 rounded-full border border-slate-700 px-2 py-1 text-xs text-slate-200">
            <span className={trend.cls}>{trend.icon}</span>
            <span>{trend.label}</span>
            <span className="text-slate-400">•</span>
            <span className="font-semibold">{statusLabel(status)}</span>
          </div>
        </div>
        <div className="text-right">
          <div className="text-xs text-slate-400">Status geral</div>
          <div className="text-sm font-semibold text-slate-100 mt-1">{statusLabel(status)}</div>
        </div>
      </div>
      <div className="mt-4 text-sm text-slate-200/90 leading-relaxed">
        {summary}
      </div>
    </div>
  );
}
