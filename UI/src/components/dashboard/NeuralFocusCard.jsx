import React from "react";

function sevColor(sev) {
  if (sev === "critical") return "from-rose-500/20 to-rose-500/5 border-rose-500/30";
  if (sev === "warning") return "from-amber-500/20 to-amber-500/5 border-amber-500/30";
  return "from-emerald-500/20 to-emerald-500/5 border-emerald-500/30";
}

function sevLabel(sev) {
  if (sev === "critical") return "Crítico";
  if (sev === "warning") return "Atenção";
  return "Estável";
}

function Skeleton() {
  return (
    <div className="animate-pulse space-y-3">
      <div className="h-4 w-28 bg-slate-800/80 rounded" />
      <div className="h-7 w-48 bg-slate-800/80 rounded" />
      <div className="h-4 w-full bg-slate-800/80 rounded" />
      <div className="h-4 w-2/3 bg-slate-800/80 rounded" />
    </div>
  );
}

export default function NeuralFocusCard({ data, loading, error }) {
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
        <div className="text-sm text-rose-200">Falha ao carregar foco neural.</div>
      </div>
    );
  }

  const issue = data?.primary_issue || {};
  const title = issue.title || "Sem problema principal";
  const summary = issue.summary || "Nenhuma anomalia crítica detectada.";
  const severity = issue.severity || "healthy";
  const impact = Number.isFinite(issue.impact_count) ? issue.impact_count : 0;

  return (
    <div className={[
      "rounded-2xl border bg-gradient-to-br p-5 shadow-lg",
      sevColor(severity),
    ].join(" ")}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="text-xs uppercase tracking-widest text-slate-300">Problema Principal</div>
        <span className="text-xs rounded-full border border-slate-700 px-2 py-1 text-slate-200">
          {sevLabel(severity)}
        </span>
      </div>
      <div className="mt-2 text-lg font-semibold text-slate-100">{title}</div>
      <div className="mt-2 text-sm text-slate-200/90">{summary}</div>
      {impact > 0 && (
        <div className="mt-3 text-xs text-slate-300">
          Impacto estimado: <span className="text-slate-100 font-semibold">{impact}</span> devices
        </div>
      )}
    </div>
  );
}
