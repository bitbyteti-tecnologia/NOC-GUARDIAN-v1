import React, { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import api from "../lib/api";
import HostDrawer from "../components/HostDrawer";

function Card({ title, value }) {
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
      <div className="text-xs text-slate-400 font-semibold">{title}</div>
      <div className="text-xl font-extrabold text-slate-100 mt-1">{value}</div>
    </div>
  );
}

function fmtDate(iso) {
  if (!iso) return "-";
  try {
    return new Intl.DateTimeFormat("pt-BR", {
      dateStyle: "short",
      timeStyle: "medium",
      timeZone: "America/Sao_Paulo",
    }).format(new Date(iso));
  } catch {
    return iso;
  }
}

function fmtPct(v) {
  const n = Number(v);
  if (!Number.isFinite(n)) return "-";
  return `${n.toFixed(2)}%`;
}

export default function Customer() {
  const params = useParams();
  const tenantId = params.tenantId || params.id || params.tenantID || params.tenant || "";

  const [summary, setSummary] = useState(null);
  const [hosts, setHosts] = useState([]);
  const [expandedHost, setExpandedHost] = useState(""); // hostname
  const [loading, setLoading] = useState(false);

  async function loadAll() {
    if (!tenantId) return;
    setLoading(true);
    try {
      const [s, h] = await Promise.all([
        api.get(`/api/v1/tenants/${tenantId}/dashboard/summary`),
        api.get(`/api/v1/tenants/${tenantId}/dashboard/hosts`),
      ]);
      setSummary(s.data || null);
      setHosts(h.data?.hosts || h.data || []);
    } catch {
      setSummary(null);
      setHosts([]);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadAll();
    // eslint-disable-next-line
  }, [tenantId]);

  const hostsSorted = useMemo(() => {
    const arr = Array.isArray(hosts) ? [...hosts] : [];
    // ordena por last_seen desc
    arr.sort((a, b) => String(b?.last_seen || "").localeCompare(String(a?.last_seen || "")));
    return arr;
  }, [hosts]);

  function toggleRow(hostname) {
    setExpandedHost((cur) => (cur === hostname ? "" : hostname));
  }

  const expandedHostObj = useMemo(
    () => hostsSorted.find((x) => x.hostname === expandedHost),
    [hostsSorted, expandedHost]
  );

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">Dashboard do Cliente</h1>
          <div className="text-xs text-slate-400 mt-1">
            Tenant: <span className="font-mono text-slate-300">{tenantId || "(vazio)"}</span>
          </div>
        </div>

        <button
          className="px-4 py-2 bg-sky-600 rounded hover:bg-sky-500 font-semibold"
          onClick={loadAll}
          disabled={loading}
        >
          {loading ? "..." : "Atualizar"}
        </button>
      </div>

      {/* cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card title="Hosts" value={summary?.total_hosts ?? "-"} />
        <Card title="Online" value={summary?.online ?? "-"} />
        <Card title="Offline" value={summary?.offline ?? "-"} />
        <Card title="Último heartbeat" value={fmtDate(summary?.last_heartbeat)} />
      </div>

      {/* downloads (mantém seu bloco atual sem mexer) */}
      <div className="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
        <div className="font-semibold text-slate-100">Downloads de Agentes</div>
        <div className="text-xs text-slate-400 mt-1">
          Instale o agente no Windows ou Linux para começar a coletar métricas.
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mt-4">
          <div className="rounded-lg border border-slate-800 bg-slate-950/60 p-3">
            <div className="text-sm font-bold">Windows (MSI)</div>
            <div className="text-xs text-slate-400 mt-1">nocguardian-agent.msi</div>
          </div>
          <div className="rounded-lg border border-slate-800 bg-slate-950/60 p-3">
            <div className="text-sm font-bold">Linux ARM64 (.deb)</div>
            <div className="text-xs text-slate-400 mt-1">nocguardian-agent_arm64.deb</div>
          </div>
          <div className="rounded-lg border border-slate-800 bg-slate-950/60 p-3">
            <div className="text-sm font-bold">Linux AMD64 (.deb)</div>
            <div className="text-xs text-slate-400 mt-1">nocguardian-agent_amd64.deb</div>
          </div>
        </div>
      </div>

      {/* HOSTS table + inline drawer */}
      <div className="rounded-xl border border-slate-800 bg-slate-950/50 overflow-hidden">
        <div className="px-4 py-3 border-b border-slate-800 font-semibold text-slate-100">
          Hosts
        </div>

        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead className="text-slate-400">
              <tr className="border-b border-slate-800">
                <th className="text-left px-4 py-3">Hostname</th>
                <th className="text-left px-4 py-3">Status</th>
                <th className="text-left px-4 py-3">CPU</th>
                <th className="text-left px-4 py-3">Mem</th>
                <th className="text-left px-4 py-3">Disco</th>
                <th className="text-left px-4 py-3">Último</th>
              </tr>
            </thead>

            <tbody className="text-slate-200">
              {hostsSorted.map((h) => {
                const isOpen = expandedHost === h.hostname;
                return (
                  <React.Fragment key={h.hostname}>
                    <tr
                      className={[
                        "border-b border-slate-800 cursor-pointer hover:bg-slate-900/40",
                        isOpen ? "bg-slate-900/30" : "",
                      ].join(" ")}
                      onClick={() => toggleRow(h.hostname)}
                      title="Clique para abrir/fechar detalhes do host"
                    >
                      <td className="px-4 py-3 font-semibold">{h.hostname}</td>
                      <td className="px-4 py-3">
                        <span className={`px-3 py-1 rounded-full text-xs font-bold border
                          ${h.status === "ONLINE"
                            ? "bg-emerald-500/15 text-emerald-200 border-emerald-500/30"
                            : "bg-amber-500/15 text-amber-200 border-amber-500/30"}`}>
                          {h.status || "-"}
                        </span>
                      </td>
                      <td className="px-4 py-3">{fmtPct(h.cpu_percent)}</td>
                      <td className="px-4 py-3">{fmtPct(h.mem_used_pct)}</td>
                      <td className="px-4 py-3">{fmtPct(h.disk_used_pct)}</td>
                      <td className="px-4 py-3 text-slate-400">{fmtDate(h.last_seen)}</td>
                    </tr>

                    {isOpen && (
                      <tr className="border-b border-slate-800">
                        <td colSpan={6} className="px-4 pb-4">
                          <HostDrawer
                            tenantId={tenantId}
                            host={expandedHostObj}
                            open={true}
                            onClose={() => setExpandedHost("")}
                            api={api}
                            variant="inline"
                          />
                        </td>
                      </tr>
                    )}
                  </React.Fragment>
                );
              })}

              {hostsSorted.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-4 py-4 text-slate-400">
                    Nenhum host encontrado.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}