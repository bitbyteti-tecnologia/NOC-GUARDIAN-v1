import React, { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import api from "../lib/api";
import useMe from "../hooks/useMe";

export default function CustomerExecutive() {
  const params = useParams();
  const navigate = useNavigate();
  const { me } = useMe();
  const tenantId = params.tenantId || params.id || params.tenantID || params.tenant || "";
  const isSuperAdmin = me?.role === "superadmin";

  const [tenantName, setTenantName] = useState("");
  const [summary, setSummary] = useState(null);
  const [alerts, setAlerts] = useState([]);
  const [hosts, setHosts] = useState([]);
  const [loading, setLoading] = useState(false);

  async function loadTenantInfo() {
    if (!tenantId) return;
    try {
      const r = await api.get(`/api/v1/tenants/${tenantId}`);
      setTenantName(r.data?.name || "");
    } catch {
      setTenantName("");
    }
  }

  async function loadExecutive() {
    if (!tenantId) return;
    setLoading(true);
    try {
      const [s, a, h] = await Promise.all([
        api.get(`/api/v1/tenants/${tenantId}/dashboard/summary`),
        api.get(`/api/v1/${tenantId}/alerts`),
        api.get(`/api/v1/tenants/${tenantId}/dashboard/hosts`),
      ]);
      setSummary(s.data || null);
      setAlerts(a.data || []);
      setHosts(h.data?.hosts || h.data || []);
    } catch {
      setSummary(null);
      setAlerts([]);
      setHosts([]);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadTenantInfo();
    loadExecutive();
    // eslint-disable-next-line
  }, [tenantId]);

  const alertStats = useMemo(() => {
    const items = Array.isArray(alerts) ? alerts : [];
    const last24h = items.filter((a) => {
      const t = new Date(a.time).getTime();
      return Date.now() - t <= 24 * 60 * 60 * 1000;
    });
    const critical = last24h.filter((a) => a.severity === "critical").length;
    const warning = last24h.filter((a) => a.severity === "warning").length;
    return { total: last24h.length, critical, warning };
  }, [alerts]);

  const topHosts = useMemo(() => {
    const arr = Array.isArray(hosts) ? [...hosts] : [];
    arr.sort((a, b) => (b.cpu_percent ?? 0) - (a.cpu_percent ?? 0));
    return arr.slice(0, 5);
  }, [hosts]);

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">Dashboard Executivo</h1>
          <div className="text-xs text-slate-400 mt-1">
            Cliente: <span className="text-slate-200">{tenantName || "(sem nome)"}</span>
            {tenantId ? (
              <span className="ml-2 text-slate-500 font-mono">({tenantId})</span>
            ) : null}
          </div>
          <div className="mt-3 inline-flex rounded-full border border-slate-700 bg-slate-900/50 p-1 text-xs">
            <Link
              to={`/tenant/${tenantId}`}
              className="rounded-full px-3 py-1 text-slate-300 hover:text-slate-100"
            >
              Padrão
            </Link>
            <span className="rounded-full px-3 py-1 bg-sky-600 text-white font-semibold">
              Executivo
            </span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {isSuperAdmin && (
            <button
              className="px-3 py-2 bg-slate-900 border border-slate-700 rounded hover:bg-slate-800 text-sm"
              onClick={() => navigate("/")}
            >
              Voltar
            </button>
          )}
          <button
            className="px-4 py-2 bg-sky-600 rounded hover:bg-sky-500 font-semibold"
            onClick={loadExecutive}
            disabled={loading}
          >
            {loading ? "..." : "Atualizar"}
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-4 gap-4">
        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="text-xs text-slate-400">Hosts monitorados</div>
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
            {alertStats.total}
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
          <div className="text-xs text-slate-400">Resumo executivo</div>
          <div className="text-sm text-slate-400 mt-2">
            Disponibilidade, incidentes críticos e tendências.
          </div>
          <button className="mt-4 px-4 py-2 bg-slate-800 rounded text-sm font-semibold hover:bg-slate-700">
            Exportar PDF
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="font-semibold text-slate-100">Top 5 hosts por CPU</div>
          <div className="text-xs text-slate-400 mt-1">
            Últimas leituras de performance.
          </div>
          <div className="mt-4 space-y-2">
            {topHosts.map((h) => (
              <div key={h.hostname} className="flex items-center justify-between text-sm">
                <span className="text-slate-200">{h.hostname}</span>
                <span className="text-sky-300 font-semibold">
                  {(h.cpu_percent ?? 0).toFixed(1)}%
                </span>
              </div>
            ))}
            {topHosts.length === 0 && (
              <div className="text-xs text-slate-500">Sem dados de CPU ainda.</div>
            )}
          </div>
        </div>

        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
          <div className="font-semibold text-slate-100">Alertas recentes</div>
          <div className="text-xs text-slate-400 mt-1">
            Últimos 5 eventos críticos e de atenção.
          </div>
          <div className="mt-4 space-y-2">
            {(alerts || []).slice(0, 5).map((a) => (
              <div key={a.id} className="text-sm border-b border-slate-800 pb-2">
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
