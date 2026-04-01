import React, { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import api from "../lib/api";

export default function Reports() {
  const { tenantID } = useParams();
  const [summary, setSummary] = useState(null);
  const [alerts, setAlerts] = useState([]);
  const [hosts, setHosts] = useState([]);

  useEffect(() => {
    if (!tenantID) return;
    api
      .get(`/api/v1/tenants/${tenantID}/dashboard/summary`)
      .then((r) => setSummary(r.data))
      .catch(() => setSummary(null));
    api
      .get(`/api/v1/${tenantID}/alerts`)
      .then((r) => setAlerts(r.data || []))
      .catch(() => setAlerts([]));
    api
      .get(`/api/v1/tenants/${tenantID}/dashboard/hosts`)
      .then((r) => setHosts(r.data?.hosts || r.data || []))
      .catch(() => setHosts([]));
  }, [tenantID]);

  const alertStats = useMemo(() => {
    const items = Array.isArray(alerts) ? alerts : [];
    const last24h = items.filter((a) => {
      const t = new Date(a.time).getTime();
      return Date.now() - t <= 24 * 60 * 60 * 1000;
    });
    const critical = last24h.filter((a) => a.severity === "critical").length;
    const warning = last24h.filter((a) => a.severity === "warning").length;
    return { total24h: last24h.length, critical, warning };
  }, [alerts]);

  const topCPU = useMemo(() => {
    const rows = Array.isArray(hosts) ? [...hosts] : [];
    rows.sort((a, b) => (b.cpu_percent ?? 0) - (a.cpu_percent ?? 0));
    return rows.slice(0, 5);
  }, [hosts]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Relatórios</h1>
        <div className="text-xs text-slate-400 mt-1">
          Tenant: <span className="text-slate-200 font-mono">{tenantID}</span>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-4 gap-4">
        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="text-xs text-slate-400">Hosts</div>
          <div className="text-2xl font-semibold text-slate-100 mt-1">
            {summary?.total_hosts ?? 0}
          </div>
          <div className="text-xs text-slate-500 mt-2">
            Online {summary?.online ?? 0} · Offline {summary?.offline ?? 0}
          </div>
        </div>

        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="text-xs text-slate-400">Alertas (24h)</div>
          <div className="text-2xl font-semibold text-slate-100 mt-1">
            {alertStats.total24h}
          </div>
          <div className="text-xs text-slate-500 mt-2">
            Críticos {alertStats.critical} · Atenção {alertStats.warning}
          </div>
        </div>

        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="text-xs text-slate-400">Último heartbeat</div>
          <div className="text-sm text-slate-100 mt-2">
            {summary?.last_any_heartbeat
              ? new Date(summary.last_any_heartbeat).toLocaleString("pt-BR")
              : "sem dados"}
          </div>
        </div>

        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="text-xs text-slate-400">Relatório executivo</div>
          <div className="text-sm text-slate-400 mt-2">
            Consolida disponibilidade e incidentes críticos.
          </div>
          <button className="mt-4 px-4 py-2 bg-sky-600 rounded text-sm font-semibold hover:bg-sky-500">
            Exportar resumo
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="font-semibold text-slate-100">Top 5 hosts por CPU</div>
          <div className="text-xs text-slate-400 mt-1">
            Baseado na última leitura de desempenho.
          </div>
          <div className="mt-4 space-y-2">
            {topCPU.map((h) => (
              <div key={h.hostname} className="flex items-center justify-between text-sm">
                <span className="text-slate-200">{h.hostname}</span>
                <span className="text-slate-400">
                  {(h.cpu_percent ?? 0).toFixed(1)}%
                </span>
              </div>
            ))}
            {topCPU.length === 0 && (
              <div className="text-xs text-slate-500">Sem dados de CPU ainda.</div>
            )}
          </div>
        </div>

        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="font-semibold text-slate-100">Alertas recentes</div>
          <div className="text-xs text-slate-400 mt-1">
            Últimos 5 eventos com maior severidade.
          </div>
          <div className="mt-4 space-y-2">
            {(alerts || []).slice(0, 5).map((a) => (
              <div key={a.id} className="text-sm">
                <div className="text-slate-200">{a.summary}</div>
                <div className="text-xs text-slate-500">
                  {a.severity} · {new Date(a.time).toLocaleString("pt-BR")}
                </div>
              </div>
            ))}
            {(alerts || []).length === 0 && (
              <div className="text-xs text-slate-500">Nenhum alerta registrado.</div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
