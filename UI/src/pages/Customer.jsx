import React, { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import api from "../lib/api";
import HostDrawer from "../components/HostDrawer";
import { computeHostSeverity } from "../features/telemetry/health";

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
  const tenantId = params.tenantId || params.id || params.tenantID || params.tenant || "";

  const [summary, setSummary] = useState(null);
  const [hosts, setHosts] = useState([]);
  const [selectedHost, setSelectedHost] = useState(""); // hostname
  const [loading, setLoading] = useState(false);
  const [statusFilter, setStatusFilter] = useState("all");
  const [severityFilter, setSeverityFilter] = useState("all");
  const [osFilter, setOsFilter] = useState("all");
  const sortKey = "last_seen";
  const sortDir = "desc";

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

  useEffect(() => {
    if (!selectedHost && hostsSorted.length) {
      setSelectedHost(hostsSorted[0].hostname);
    }
  }, [hostsSorted, selectedHost]);

  const selectedHostObj = useMemo(
    () => hostsSorted.find((x) => x.hostname === selectedHost),
    [hostsSorted, selectedHost]
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

      {/* Seleção de host + resumo minimalista */}
      <div className="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3">
          <div>
            <div className="text-sm font-semibold text-slate-100">Host selecionado</div>
            <div className="text-xs text-slate-400 mt-1">
              Último heartbeat: <span className="text-slate-200">{fmtDate(lastHeartbeat)}</span>
            </div>
          </div>
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
            <select
              value={selectedHost}
              onChange={(e) => setSelectedHost(e.target.value)}
              className="bg-slate-950/70 border border-slate-700 rounded px-2 py-1"
            >
              {hostsSorted.map((h) => (
                <option key={h.hostname} value={h.hostname}>
                  {h.hostname}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>

      {/* Telemetria do host selecionado */}
      {selectedHostObj ? (
        <HostDrawer
          tenantId={tenantId}
          host={selectedHostObj}
          open={true}
          onClose={() => {}}
          api={api}
          variant="inline"
        />
      ) : (
        <div className="text-sm text-slate-400">Nenhum host encontrado.</div>
      )}
    </div>
  );
}
