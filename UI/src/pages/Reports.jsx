import React from "react";
import { useParams } from "react-router-dom";

export default function Reports() {
  const { tenantID } = useParams();
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Relatórios</h1>
        <div className="text-xs text-slate-400 mt-1">
          Tenant: <span className="text-slate-200 font-mono">{tenantID}</span>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="font-semibold text-slate-100">Executivo</div>
          <div className="text-xs text-slate-400 mt-2">
            Resumo de alto nível para gerência: disponibilidade, incidentes críticos e tendências.
          </div>
          <button className="mt-4 px-4 py-2 bg-sky-600 rounded text-sm font-semibold hover:bg-sky-500">
            Gerar relatório
          </button>
        </div>

        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="font-semibold text-slate-100">SLA & Disponibilidade</div>
          <div className="text-xs text-slate-400 mt-2">
            Percentual de uptime e cumprimento de metas contratuais por período.
          </div>
          <button className="mt-4 px-4 py-2 bg-sky-600 rounded text-sm font-semibold hover:bg-sky-500">
            Gerar relatório
          </button>
        </div>

        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="font-semibold text-slate-100">KPIs de Performance</div>
          <div className="text-xs text-slate-400 mt-2">
            Latência, perda de pacotes, CPU e memória por host.
          </div>
          <button className="mt-4 px-4 py-2 bg-sky-600 rounded text-sm font-semibold hover:bg-sky-500">
            Gerar relatório
          </button>
        </div>

        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="font-semibold text-slate-100">Consumo & Billing</div>
          <div className="text-xs text-slate-400 mt-2">
            Previsão de custos por ativo e por período.
          </div>
          <button className="mt-4 px-4 py-2 bg-sky-600 rounded text-sm font-semibold hover:bg-sky-500">
            Gerar relatório
          </button>
        </div>
      </div>
    </div>
  );
}

