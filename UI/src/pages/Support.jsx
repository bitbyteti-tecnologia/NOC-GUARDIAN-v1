import React from "react";
import { useParams } from "react-router-dom";

export default function Support() {
  const { tenantID } = useParams();
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Suporte & Tickets</h1>
        <div className="text-xs text-slate-400 mt-1">
          Tenant: <span className="text-slate-200 font-mono">{tenantID}</span>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="font-semibold text-slate-100">Abrir chamado</div>
          <div className="text-xs text-slate-400 mt-2">
            Envie uma solicitação de suporte diretamente para a equipe.
          </div>
          <button className="mt-4 px-4 py-2 bg-sky-600 rounded text-sm font-semibold hover:bg-sky-500">
            Novo chamado
          </button>
        </div>

        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="font-semibold text-slate-100">Meus chamados</div>
          <div className="text-xs text-slate-400 mt-2">
            Acompanhe o status e o histórico das solicitações anteriores.
          </div>
          <button className="mt-4 px-4 py-2 bg-slate-800 rounded text-sm font-semibold hover:bg-slate-700">
            Ver chamados
          </button>
        </div>
      </div>
    </div>
  );
}

