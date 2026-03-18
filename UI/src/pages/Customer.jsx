import React, { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import api from "../lib/api";
import HostDrawer from "../components/HostDrawer";
import { computeHostSeverity } from "../features/telemetry/health";

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
  const [statusFilter, setStatusFilter] = useState("all");
  const [severityFilter, setSeverityFilter] = useState("all");
  const [osFilter, setOsFilter] = useState("all");
  const [sortKey, setSortKey] = useState("last_seen");
  const [sortDir, setSortDir] = useState("desc");

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

  const enrichedHosts = useMemo(() => {
    const arr = Array.isArray(hosts) ? hosts.map((h) => ({ ...h })) : [];
    arr.forEach((h) => {
      h._severity = computeHostSeverity(h);
    });
    return arr;
  }, [hosts]);

  const agg = useMemo(() => {
    const arr = enrichedHosts;
    const total = arr.length || 0;
    if (!total) {
      return {
        highCpuPct: "-",
        highMemPct: "-",
        highDiskPct: "-",
      };
    }
    const highCpu = arr.filter((h) => Number(h.cpu_percent) >= 80).length;
    const highMem = arr.filter((h) => Number(h.mem_used_pct) >= 80).length;
    const highDisk = arr.filter((h) => Number(h.disk_used_pct) >= 90).length;
    const toPct = (n) => `${((n / total) * 100).toFixed(0)}%`;
    return {
      highCpuPct: toPct(highCpu),
      highMemPct: toPct(highMem),
      highDiskPct: toPct(highDisk),
    };
  }, [enrichedHosts]);

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
        <Card title="Último heartbeat" value={fmtDate(summary?.last_any_heartbeat)} />
      </div>

      {/* cards de risco */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card title="% Hosts CPU ≥ 80%" value={agg.highCpuPct} />
        <Card title="% Hosts Mem ≥ 80%" value={agg.highMemPct} />
        <Card title="% Hosts Disco ≥ 90%" value={agg.highDiskPct} />
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
                <th className="text-left px-4 py-3 cursor-pointer" onClick={() => changeSort("cpu_percent")}>
                  CPU
                </th>
                <th className="text-left px-4 py-3 cursor-pointer" onClick={() => changeSort("mem_used_pct")}>
                  Mem
                </th>
                <th className="text-left px-4 py-3 cursor-pointer" onClick={() => changeSort("disk_used_pct")}>
                  Disco
                </th>
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
                      <td className="px-4 py-3">{fmtPct(h.cpu_percent)}</td>
                      <td className="px-4 py-3">{fmtPct(h.mem_used_pct)}</td>
                      <td className="px-4 py-3">{fmtPct(h.disk_used_pct)}</td>
                      <td className="px-4 py-3 text-slate-400">{fmtDate(h.last_seen)}</td>
                    </tr>

                    {isOpen && (
                      <tr className="border-b border-slate-800">
                        <td colSpan={9} className="px-4 pb-4">
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