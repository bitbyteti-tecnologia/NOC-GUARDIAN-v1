import React, { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import api from "../lib/api";

export default function Support() {
  const { tenantID } = useParams();
  const [alerts, setAlerts] = useState([]);

  useEffect(() => {
    if (!tenantID) return;
    api
      .get(`/api/v1/${tenantID}/alerts`)
      .then((r) => setAlerts(r.data || []))
      .catch(() => setAlerts([]));
  }, [tenantID]);

  const activeAlerts = useMemo(() => {
    const items = Array.isArray(alerts) ? alerts : [];
    return items.filter((a) => a.status !== "resolved");
  }, [alerts]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Suporte & Tickets</h1>
        <div className="text-xs text-slate-400 mt-1">
          Tenant: <span className="text-slate-200 font-mono">{tenantID}</span>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="text-xs text-slate-400">Alertas ativos</div>
          <div className="text-2xl font-semibold text-slate-100 mt-1">
            {activeAlerts.length}
          </div>
          <div className="text-xs text-slate-500 mt-2">
            Eventos não resolvidos que podem virar tickets.
          </div>
        </div>

        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="font-semibold text-slate-100">Abrir chamado</div>
          <div className="text-xs text-slate-400 mt-2">
            Use um alerta ativo como base para abrir uma solicitação.
          </div>
          <button className="mt-4 px-4 py-2 bg-sky-600 rounded text-sm font-semibold hover:bg-sky-500">
            Novo chamado
          </button>
        </div>

        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="font-semibold text-slate-100">Histórico</div>
          <div className="text-xs text-slate-400 mt-2">
            Últimos eventos e solicitações associadas.
          </div>
          <button className="mt-4 px-4 py-2 bg-slate-800 rounded text-sm font-semibold hover:bg-slate-700">
            Ver histórico
          </button>
        </div>
      </div>

      <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
        <div className="font-semibold text-slate-100">Alertas recentes</div>
        <div className="text-xs text-slate-400 mt-1">
          Use esta lista para abrir tickets rapidamente.
        </div>
        <div className="mt-4 space-y-2">
          {activeAlerts.slice(0, 8).map((a) => (
            <div key={a.id} className="text-sm border-b border-slate-800 pb-2">
              <div className="text-slate-200">{a.summary}</div>
              <div className="text-xs text-slate-500">
                {a.severity} · {new Date(a.time).toLocaleString("pt-BR")}
              </div>
            </div>
          ))}
          {activeAlerts.length === 0 && (
            <div className="text-xs text-slate-500">Nenhum alerta ativo no momento.</div>
          )}
        </div>
      </div>
    </div>
  );
}
