import React from "react";
import { useParams } from "react-router-dom";

export default function Inventory() {
  const { tenantID } = useParams();
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Inventário</h1>
        <div className="text-xs text-slate-400 mt-1">
          Tenant: <span className="text-slate-200 font-mono">{tenantID}</span>
        </div>
      </div>

      <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
        <div className="font-semibold text-slate-100">Hardware, Software e Licenças</div>
        <div className="text-xs text-slate-400 mt-2">
          Esta página consolida ativos, versões e licenças expirando. Integração em andamento.
        </div>
      </div>
    </div>
  );
}

