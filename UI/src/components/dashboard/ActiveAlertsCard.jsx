import React, { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import api from "../../lib/api";

export default function ActiveAlertsCard({ tenantId }) {
  const [alerts, setAlerts] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (!tenantId) return;
    setLoading(true);
    setError(false);
    api
      .get(`/api/v1/${tenantId}/alerts`)
      .then((r) => setAlerts(r.data || []))
      .catch(() => {
        setAlerts([]);
        setError(true);
      })
      .finally(() => setLoading(false));
  }, [tenantId]);

  const active = useMemo(() => {
    const arr = Array.isArray(alerts) ? alerts : [];
    return arr.filter((a) => String(a.status || "").toLowerCase() !== "resolved").slice(0, 5);
  }, [alerts]);

  return (
    <div className="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
      <div className="flex items-start justify-between gap-3 mb-3">
        <div>
          <div className="font-semibold text-slate-100">Alertas Ativos</div>
          <div className="text-xs text-slate-400 mt-1">
            Últimos alertas não resolvidos do tenant.
          </div>
        </div>
        <Link
          to={`/tenant/${tenantId}/alerts`}
          className="text-xs text-sky-300 hover:text-sky-200"
        >
          Ver todos
        </Link>
      </div>

      {loading && <div className="text-xs text-slate-500">Carregando...</div>}
      {error && <div className="text-xs text-amber-300">Falha ao carregar alertas.</div>}

      <div className="space-y-2">
        {active.length === 0 && !loading && (
          <div className="text-xs text-slate-500">Nenhum alerta ativo.</div>
        )}
        {active.map((a) => (
          <div key={a.id} className="flex items-center justify-between text-sm border-b border-slate-800 pb-2">
            <div>
              <div className="text-slate-200 font-semibold">{a.summary || "Alerta"}</div>
              <div className="text-xs text-slate-500">
                {a.severity || "info"} • {new Date(a.time).toLocaleString()}
              </div>
            </div>
            <span
              className={`text-xs px-2 py-1 rounded-full ${
                a.severity === "critical"
                  ? "bg-rose-500/20 text-rose-200 border border-rose-500/40"
                  : "bg-amber-500/20 text-amber-200 border border-amber-500/40"
              }`}
            >
              {a.status || "aberto"}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

