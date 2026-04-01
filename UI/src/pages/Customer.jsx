import React, { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import api from "../lib/api";
import HostDrawer from "../components/HostDrawer";
import { computeHostSeverity } from "../features/telemetry/health";
import { useTelemetryFromApi } from "../features/telemetry/integrations/useTelemetryFromApi";
import { LanBandwidthCard } from "../features/telemetry/components/LanBandwidthCard";
import { WanPerformanceCard } from "../features/telemetry/components/WanPerformanceCard";
import HealthCard from "../components/dashboard/HealthCard";
import IncidentsCard from "../components/dashboard/IncidentsCard";
import InsightsCard from "../components/dashboard/InsightsCard";
import RecommendationsCard from "../components/dashboard/RecommendationsCard";
import IncidentDrawer from "../components/dashboard/IncidentDrawer";
import TopologyCard from "../components/dashboard/TopologyCard";
import TopServicesCard from "../components/dashboard/TopServicesCard";
import useMe from "../hooks/useMe";

function fmtDate(iso) {
  if (!iso) return "-";
  try {
    if (iso instanceof Date) {
      return new Intl.DateTimeFormat("pt-BR", {
        dateStyle: "short",
        timeStyle: "medium",
        timeZone: "America/Sao_Paulo",
      }).format(iso);
    }
    if (typeof iso === "number") {
      const ms = iso > 10_000_000_000 ? iso : iso * 1000;
      return new Intl.DateTimeFormat("pt-BR", {
        dateStyle: "short",
        timeStyle: "medium",
        timeZone: "America/Sao_Paulo",
      }).format(new Date(ms));
    }
    return new Intl.DateTimeFormat("pt-BR", {
      dateStyle: "short",
      timeStyle: "medium",
      timeZone: "America/Sao_Paulo",
    }).format(new Date(iso));
  } catch {
    return iso;
  }
}

export default function Customer() {
  const params = useParams();
  const navigate = useNavigate();
  const { me } = useMe();
  const tenantId = params.tenantId || params.id || params.tenantID || params.tenant || "";
  const isSuperAdmin = me?.role === "superadmin";

  const [summary, setSummary] = useState(null);
  const [hosts, setHosts] = useState([]);
  const [expandedHost, setExpandedHost] = useState(""); // hostname
  const [tenantName, setTenantName] = useState("");
  const [loading, setLoading] = useState(false);
  const [statusFilter, setStatusFilter] = useState("all");
  const [severityFilter, setSeverityFilter] = useState("all");
  const [osFilter, setOsFilter] = useState("all");
  const [sortKey, setSortKey] = useState("last_seen");
  const [sortDir, setSortDir] = useState("desc");
  const [intel, setIntel] = useState(null);
  const [intelLoading, setIntelLoading] = useState(false);
  const [intelError, setIntelError] = useState(false);
  const [incidentOpen, setIncidentOpen] = useState(false);
  const [incidentLoading, setIncidentLoading] = useState(false);
  const [incidentError, setIncidentError] = useState(false);
  const [incidentDetails, setIncidentDetails] = useState(null);
  const [scanIPs, setScanIPs] = useState("");
  const [scanCommunity, setScanCommunity] = useState("");
  const [useSNMPScan, setUseSNMPScan] = useState(false);
  const [scanLoading, setScanLoading] = useState(false);
  const [scanMsg, setScanMsg] = useState("");
  const [topology, setTopology] = useState(null);
  const [topologyLoading, setTopologyLoading] = useState(false);
  const [topologyError, setTopologyError] = useState(false);
  const downloads = [
    { label: "Windows (MSI)", file: "nocguardian-agent.msi" },
    { label: "Linux ARM64 (.deb)", file: "nocguardian-agent_arm64.deb" },
    { label: "Linux AMD64 (.deb)", file: "nocguardian-agent_amd64.deb" },
    { label: "Linux aarch64 (.rpm)", file: "nocguardian-agent_aarch64.rpm" },
    { label: "Linux x86_64 (.rpm)", file: "nocguardian-agent_x86_64.rpm" },
  ];

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

  async function loadTenantInfo() {
    if (!tenantId) return;
    try {
      const r = await api.get(`/api/v1/tenants/${tenantId}`);
      setTenantName(r.data?.name || "");
    } catch {
      setTenantName("");
    }
  }

  async function loadIntelligence() {
    if (!tenantId) return;
    setIntelLoading(true);
    setIntelError(false);
    try {
      const r = await api.get(`/api/v1/dashboard/intelligence`, {
        headers: {
          "X-Tenant-Id": tenantId,
        },
      });
      setIntel(r.data || null);
    } catch {
      setIntel(null);
      setIntelError(true);
    } finally {
      setIntelLoading(false);
    }
  }

  async function openIncident(inc) {
    if (!inc?.incident_id || !tenantId) return;
    setIncidentOpen(true);
    setIncidentLoading(true);
    setIncidentError(false);
    try {
      const r = await api.get(`/api/v1/dashboard/incidents/${encodeURIComponent(inc.incident_id)}/details`, {
        headers: {
          "X-Tenant-Id": tenantId,
        },
      });
      setIncidentDetails(r.data || null);
    } catch {
      setIncidentDetails(null);
      setIncidentError(true);
    } finally {
      setIncidentLoading(false);
    }
  }

  async function loadTopology() {
    if (!tenantId) return;
    setTopologyLoading(true);
    setTopologyError(false);
    try {
      const r = await api.get(`/api/v1/dashboard/topology`, {
        headers: {
          "X-Tenant-Id": tenantId,
        },
      });
      setTopology(r.data || null);
    } catch {
      setTopology(null);
      setTopologyError(true);
    } finally {
      setTopologyLoading(false);
    }
  }

  async function runDiscovery() {
    if (!tenantId) return;
    setScanLoading(true);
    setScanMsg("");
    const ips = String(scanIPs || "")
      .split(/[\n,;\s]+/g)
      .map((v) => v.trim())
      .filter(Boolean);
    const payload = {
      ips,
      snmp: useSNMPScan && scanCommunity
        ? { version: "v2c", community: scanCommunity }
        : null,
    };
    try {
      await api.post(`/api/v1/tenants/${tenantId}/discovery`, payload);
      setScanMsg("Discovery iniciado. Aguarde alguns minutos.");
      loadTopology();
      loadAll();
    } catch {
      setScanMsg("Falha ao iniciar discovery. Verifique logs e configuração.");
    } finally {
      setScanLoading(false);
    }
  }

  useEffect(() => {
    loadAll();
    loadTenantInfo();
    loadIntelligence();
    loadTopology();
    // eslint-disable-next-line
  }, [tenantId]);

  const lastHeartbeat = useMemo(() => {
    if (summary?.last_any_heartbeat) return summary.last_any_heartbeat;
    const times = (Array.isArray(hosts) ? hosts : [])
      .map((h) => h?.last_seen)
      .filter(Boolean)
      .map((t) => Date.parse(String(t)))
      .filter((n) => Number.isFinite(n));
    if (!times.length) return null;
    return new Date(Math.max(...times));
  }, [summary, hosts]);

  const hostsSorted = useMemo(() => {
    const arr = Array.isArray(hosts) ? [...hosts] : [];
    // enriquece com severidade
    arr.forEach((h) => {
      h._severity = computeHostSeverity(h);
    });

    const filtered = arr.filter((h) => {
      if (statusFilter !== "all" && h.status !== statusFilter) return false;
      if (severityFilter !== "all" && h._severity !== severityFilter) return false;
      if (osFilter !== "all") {
        const os = String(h.os || "").toLowerCase();
        if (osFilter === "linux" && !os.includes("linux")) return false;
        if (osFilter === "windows" && !os.includes("windows")) return false;
      }
      return true;
    });

    const dir = sortDir === "asc" ? 1 : -1;
    filtered.sort((a, b) => {
      const av = a[sortKey];
      const bv = b[sortKey];
      if (sortKey === "last_seen") {
        return dir * String(bv || "").localeCompare(String(av || ""));
      }
      if (sortKey === "cpu_percent" || sortKey === "mem_used_pct" || sortKey === "disk_used_pct") {
        const an = Number(av);
        const bn = Number(bv);
        if (!Number.isFinite(an) && !Number.isFinite(bn)) return 0;
        if (!Number.isFinite(an)) return 1;
        if (!Number.isFinite(bn)) return -1;
        return dir * (bn - an);
      }
      if (sortKey === "hostname") {
        return dir * String(av || "").localeCompare(String(bv || ""));
      }
      if (sortKey === "_severity") {
        const order = { critical: 2, warning: 1, ok: 0 };
        return dir * ((order[b._severity] || 0) - (order[a._severity] || 0));
      }
      return 0;
    });

    return filtered;
  }, [hosts, statusFilter, severityFilter, osFilter, sortKey, sortDir]);

  function toggleRow(hostname) {
    setExpandedHost((cur) => (cur === hostname ? "" : hostname));
  }

  function changeSort(key) {
    setSortKey((curKey) => {
      if (curKey === key) {
        setSortDir((curDir) => (curDir === "asc" ? "desc" : "asc"));
        return curKey;
      }
      setSortDir("desc");
      return key;
    });
  }

  const expandedHostObj = useMemo(
    () => hostsSorted.find((x) => x.hostname === expandedHost),
    [hostsSorted, expandedHost]
  );

  const tenantTelemetryHost = useMemo(() => {
    const arr = Array.isArray(hostsSorted) ? hostsSorted : [];
    if (!arr.length) return null;
    const online = arr.filter((h) => h.status === "ONLINE");
    if (online.length) return online[0];
    return arr[0];
  }, [hostsSorted]);

  const { vm: lanVM } = useTelemetryFromApi({
    api,
    tenantId,
    host: tenantTelemetryHost,
    window: "24h",
    enabled: Boolean(tenantTelemetryHost),
    pollMs: 60000,
  });

  const { vm: wanVM } = useTelemetryFromApi({
    api,
    tenantId,
    host: tenantTelemetryHost,
    window: "30d",
    enabled: Boolean(tenantTelemetryHost),
    pollMs: 300000,
  });

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">Dashboard do Cliente</h1>
          <div className="text-xs text-slate-400 mt-1">
            Cliente: <span className="text-slate-200">{tenantName || "(sem nome)"}</span>
            {tenantId ? (
              <span className="ml-2 text-slate-500 font-mono">({tenantId})</span>
            ) : null}
          </div>
          <div className="text-xs text-slate-400 mt-1">
            Último heartbeat: <span className="text-slate-200">{fmtDate(lastHeartbeat)}</span>
          </div>
          <div className="mt-3 inline-flex rounded-full border border-slate-700 bg-slate-900/50 p-1 text-xs">
            <span className="rounded-full px-3 py-1 bg-sky-600 text-white font-semibold">
              Padrão
            </span>
            <Link
              to={`/tenant/${tenantId}/neural`}
              className="rounded-full px-3 py-1 text-slate-300 hover:text-slate-100"
            >
              Neural
            </Link>
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
            onClick={loadAll}
            disabled={loading}
          >
            {loading ? "..." : "Atualizar"}
          </button>
        </div>
      </div>

      {/* downloads (mantém seu bloco atual sem mexer) */}
      <div className="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
        <div className="font-semibold text-slate-100">Downloads de Agentes</div>
        <div className="text-xs text-slate-400 mt-1">
          Instale o agente no Windows ou Linux para começar a coletar métricas.
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mt-4">
          {downloads.map((d) => (
            <a
              key={d.file}
              href={`/downloads/${d.file}`}
              className="rounded-lg border border-slate-800 bg-slate-950/60 p-3 hover:bg-slate-900 transition"
              download
            >
              <div className="text-sm font-bold">{d.label}</div>
              <div className="text-xs text-slate-400 mt-1">{d.file}</div>
            </a>
          ))}
        </div>
      </div>

      {/* Discovery manual para clientes existentes */}
      <div id="discovery" className="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
        <div className="flex items-start justify-between gap-3 mb-4">
          <div>
            <div className="font-semibold text-slate-100">Discovery de Rede</div>
            <div className="text-xs text-slate-400 mt-1">
              Informe IPs e inicie o scan para popular devices e topologia.
            </div>
            {scanLoading && (
              <div className="mt-3 flex items-center gap-3 text-xs text-slate-300">
                <div className="relative h-7 w-7">
                  <div className="absolute inset-0 rounded-full border border-sky-500/40 animate-ping" />
                  <div className="absolute inset-0 rounded-full border border-sky-400/60" />
                  <svg
                    className="absolute inset-0 m-auto h-4 w-4 text-sky-300 animate-spin"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                  >
                    <circle cx="11" cy="11" r="6" />
                    <line x1="16" y1="16" x2="21" y2="21" />
                  </svg>
                </div>
                <div className="flex items-center gap-2">
                  <span>Escaneando rede...</span>
                  <div className="h-1 w-24 rounded-full bg-slate-800 overflow-hidden">
                    <div className="h-full w-1/2 bg-sky-500/60 animate-pulse" />
                  </div>
                </div>
              </div>
            )}
          </div>
          <button
            className="px-3 py-2 bg-slate-900 border border-slate-700 rounded hover:bg-slate-800 text-xs"
            onClick={runDiscovery}
            disabled={scanLoading}
          >
            {scanLoading ? "Iniciando..." : "Iniciar scan"}
          </button>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
          <div>
            <label className="text-xs text-slate-400">IPs ou CIDR</label>
            <textarea
              className="w-full p-2 rounded text-slate-900 min-h-[80px]"
              value={scanIPs}
              onChange={(e) => setScanIPs(e.target.value)}
              placeholder="Ex: 10.0.0.1, 10.0.0.2, 192.168.1.0/24"
            />
          </div>
          <div>
            <div className="flex items-center justify-between">
              <label className="text-xs text-slate-400">SNMP Community (v2c)</label>
              <label className="flex items-center gap-2 text-xs text-slate-300">
                <input
                  type="checkbox"
                  checked={useSNMPScan}
                  onChange={(e) => setUseSNMPScan(e.target.checked)}
                />
                Usar SNMP
              </label>
            </div>
            <input
              className="w-full p-2 rounded text-slate-900"
              value={scanCommunity}
              onChange={(e) => setScanCommunity(e.target.value)}
              placeholder="public"
              disabled={!useSNMPScan}
            />
            <div className="text-xs text-slate-500 mt-2">
              Sem SNMP, o discovery faz apenas seed de devices.
            </div>
          </div>
        </div>

        {scanMsg && <div className="text-xs text-slate-300 mt-3">{scanMsg}</div>}
      </div>

      {/* Inteligência do Tenant */}
      <div className="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
        <div className="flex items-start justify-between gap-3 mb-4">
          <div>
            <div className="font-semibold text-slate-100">Inteligência</div>
            <div className="text-xs text-slate-400 mt-1">
              Insights e priorização automática para decisões rápidas.
            </div>
          </div>
          <button
            className="px-3 py-2 bg-slate-900 border border-slate-700 rounded hover:bg-slate-800 text-xs"
            onClick={loadIntelligence}
            disabled={intelLoading}
          >
            {intelLoading ? "..." : "Atualizar"}
          </button>
        </div>

        <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
          <div className="xl:col-span-2">
            <HealthCard data={intel} loading={intelLoading} error={intelError} />
          </div>
          <IncidentsCard items={intel?.top_incidents || []} loading={intelLoading} error={intelError} onSelect={openIncident} />
          <InsightsCard items={intel?.insights || []} loading={intelLoading} error={intelError} />
          <RecommendationsCard items={intel?.recommendations || []} loading={intelLoading} error={intelError} />
        </div>
      </div>

      {/* Topologia do cliente */}
      <div id="topologia" className="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
        <div className="flex items-start justify-between gap-3 mb-4">
          <div>
            <div className="font-semibold text-slate-100">Topologia</div>
            <div className="text-xs text-slate-400 mt-1">
              Conexões entre dispositivos e identificação de causa raiz.
            </div>
          </div>
          <button
            className="px-3 py-2 bg-slate-900 border border-slate-700 rounded hover:bg-slate-800 text-xs"
            onClick={loadTopology}
            disabled={topologyLoading}
          >
            {topologyLoading ? "..." : "Atualizar"}
          </button>
        </div>
        <TopologyCard data={topology} loading={topologyLoading} error={topologyError} />
      </div>

      {/* Bloco de telemetria WAN/LAN do tenant */}
      <div className="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
        <div className="flex items-start justify-between gap-3 mb-4">
          <div>
            <div className="font-semibold text-slate-100">Telemetria de Rede</div>
            <div className="text-xs text-slate-400 mt-1">
              Baseado no host{" "}
              <span className="text-slate-200 font-semibold">
                {tenantTelemetryHost?.hostname || "não selecionado"}
              </span>
              .
            </div>
          </div>
          <div className="text-xs text-slate-500">
            Janelas: LAN 24h | WAN 30d
          </div>
        </div>
        <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
          <LanBandwidthCard series={lanVM?.lan?.series} />
          <WanPerformanceCard series={wanVM?.wan?.series} />
        </div>
      </div>

      {/* Top serviços do host (CPU/Mem) */}
      <TopServicesCard tenantId={tenantId} hostname={tenantTelemetryHost?.hostname || ""} />

      <IncidentDrawer
        open={incidentOpen}
        onClose={() => setIncidentOpen(false)}
        loading={incidentLoading}
        error={incidentError}
        data={incidentDetails}
      />

      {/* HOSTS table + inline drawer */}
      <div className="rounded-xl border border-slate-800 bg-slate-950/50 overflow-hidden">
        <div className="px-4 py-3 border-b border-slate-800 flex flex-col md:flex-row md:items-center md:justify-between gap-3">
          <div className="font-semibold text-slate-100">Hosts</div>
          <div className="flex flex-wrap gap-2 text-xs">
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="bg-slate-950/70 border border-slate-700 rounded px-2 py-1"
            >
              <option value="all">Todos status</option>
              <option value="ONLINE">Online</option>
              <option value="OFFLINE">Offline</option>
            </select>
            <select
              value={severityFilter}
              onChange={(e) => setSeverityFilter(e.target.value)}
              className="bg-slate-950/70 border border-slate-700 rounded px-2 py-1"
            >
              <option value="all">Todas severidades</option>
              <option value="critical">Crítico</option>
              <option value="warning">Atenção</option>
              <option value="ok">OK</option>
            </select>
            <select
              value={osFilter}
              onChange={(e) => setOsFilter(e.target.value)}
              className="bg-slate-950/70 border border-slate-700 rounded px-2 py-1"
            >
              <option value="all">Todos OS</option>
              <option value="linux">Linux</option>
              <option value="windows">Windows</option>
            </select>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead className="text-slate-400">
              <tr className="border-b border-slate-800">
                <th className="text-left px-4 py-3 cursor-pointer" onClick={() => changeSort("hostname")}>
                  Hostname
                </th>
                <th className="text-left px-4 py-3">Status</th>
                <th className="text-left px-4 py-3">Saúde</th>
                <th className="text-left px-4 py-3 hidden md:table-cell">IP</th>
                <th className="text-left px-4 py-3 hidden md:table-cell">OS</th>
                <th className="text-left px-4 py-3 cursor-pointer" onClick={() => changeSort("last_seen")}>
                  Último
                </th>
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
                      <td className="px-4 py-3">
                        <span className="text-xs font-semibold">
                          {computeHostSeverity(h) === "critical"
                            ? "CRÍTICO"
                            : computeHostSeverity(h) === "warning"
                            ? "ATENÇÃO"
                            : "OK"}
                        </span>
                      </td>
                      <td className="px-4 py-3 hidden md:table-cell text-slate-300">
                        {h.ip || h.ip_address || "-"}
                      </td>
                      <td className="px-4 py-3 hidden md:table-cell text-slate-300">
                        {h.os || "-"}
                      </td>
                      <td className="px-4 py-3 text-slate-400">{fmtDate(h.last_seen)}</td>
                    </tr>

                    {isOpen && (
                      <tr className="border-b border-slate-800">
                        <td colSpan={7} className="px-4 pb-4">
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
