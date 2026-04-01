import React, { useEffect, useMemo, useState } from "react";
import { TelemetryDashboard } from "../features/telemetry/TelemetryDashboard";
import { useTelemetryFromApi } from "../features/telemetry/integrations/useTelemetryFromApi";
import { buildHostHealthSummary, computeHostSeverity, severityBadgeClasses, severityLabel } from "../features/telemetry/health";
import TopServicesCard from "./dashboard/TopServicesCard";

function Badge({ status }) {
  const ok = status === "ONLINE";
  return (
    <span
      className={[
        "px-3 py-1 rounded-full text-xs font-bold border",
        ok
          ? "bg-emerald-500/15 text-emerald-200 border-emerald-500/30"
          : "bg-amber-500/15 text-amber-200 border-amber-500/30",
      ].join(" ")}
    >
      {status || "-"}
    </span>
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
    return String(iso);
  }
}

function fmtPct(v) {
  const n = Number(v);
  if (!Number.isFinite(n)) return "-";
  return `${n.toFixed(2)}%`;
}

function pick(obj, getters) {
  for (const g of getters) {
    try {
      const v = g(obj);
      if (v !== undefined && v !== null && v !== "") return v;
    } catch {}
  }
  return undefined;
}

export default function HostDrawer({
  tenantId,
  host,
  open,
  onClose, // mantido por compatibilidade (não usamos botão)
  api,
  variant = "inline",
}) {
  const hostname = host?.hostname || "";
  const status = host?.status || "";
  const lastSeen = host?.last_seen;

  const [hostMeta, setHostMeta] = useState(null);
  const [windowKey, setWindowKey] = useState("1h");

  const WINDOW_OPTIONS = useMemo(
    () => [
      { label: "Tempo real", value: "5m", pollMs: 5000 },
      { label: "30 min", value: "30m", pollMs: 10000 },
      { label: "1h", value: "1h", pollMs: 15000 },
      { label: "24h", value: "24h", pollMs: 60000 },
    ],
    []
  );

  const windowCfg = useMemo(() => {
    const found = WINDOW_OPTIONS.find((o) => o.value === windowKey);
    return found || WINDOW_OPTIONS[2]; // default 1h
  }, [WINDOW_OPTIONS, windowKey]);

  // Carrega IP/OS/Uptime via endpoint real do dashboard-api (inventário latest)
  useEffect(() => {
    let cancelled = false;

    async function loadHostMeta() {
      if (!open || !tenantId || !hostname || !api) return;

      const url = `/api/v1/tenants/${tenantId}/dashboard/host/${encodeURIComponent(
        hostname
      )}/inventory/latest`;

      try {
        const r = await api.get(url);
        const d = r?.data;

        const ip = pick(d, [
          (x) => x?.ip,
          (x) => x?.system?.ip,
          (x) => x?.inventory?.ip,
          (x) => x?.net?.ip,
          (x) => x?.host_ip,
          (x) => x?.ip_address,
        ]);

        const os = pick(d, [
          (x) => x?.os,
          (x) => x?.system?.os,
          (x) => x?.system?.distro,
          (x) => x?.system?.platform,
          (x) => x?.platform,
          (x) => x?.os_name,
          (x) => x?.system?.os_name,
        ]);

        const uptime = pick(d, [
          (x) => x?.uptime,
          (x) => x?.system?.uptime,
          (x) => x?.uptime_human,
        ]);

        if (!cancelled) setHostMeta({ ip, os, uptime });
      } catch {
        if (!cancelled) setHostMeta(null);
      }
    }

    loadHostMeta();
    return () => {
      cancelled = true;
    };
  }, [open, tenantId, hostname, api]);

  const hostFull = useMemo(() => (hostMeta ? { ...host, ...hostMeta } : host), [host, hostMeta]);
  const severity = computeHostSeverity(hostFull || host || {});

  // VM do painel (dados reais + polling)
  const { vm: telemetryVM } = useTelemetryFromApi({
    api,
    tenantId,
    host: hostFull,
    window: windowCfg.value,
    enabled: open,
    pollMs: windowCfg.pollMs,
  });

  if (!open) return null;

  const wrapperCls =
    variant === "inline"
      ? "rounded-xl border border-slate-800 bg-slate-950/50 p-4 mt-3"
      : "p-4";

  return (
    <div className={wrapperCls}>
      {/* Cabeçalho */}
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <div className="text-lg font-extrabold truncate">{hostname || "Host"}</div>
            <Badge status={status} />
          </div>

          <div className="text-xs text-slate-500 mt-1">
            Último: <span className="text-slate-300">{fmtDate(lastSeen)}</span>
          </div>

          <div className="text-xs text-slate-500 mt-1">
            Servidor: <span className="text-slate-300">{hostname || "-"}</span>
            {" | "}IP:{" "}
            <span className="text-slate-300">
              {hostFull?.ip || hostFull?.ip_address || hostFull?.ipAddress || hostFull?.address || "-"}
            </span>
            {" | "}OS:{" "}
            <span className="text-slate-300">
              {hostFull?.os || hostFull?.platform || hostFull?.system || hostFull?.os_name || "-"}
            </span>
            {hostFull?.uptime ? (
              <>
                {" | "}Uptime: <span className="text-slate-300">{String(hostFull.uptime)}</span>
              </>
            ) : null}
          </div>
        </div>

        {/* Indicador de saúde geral */}
        <div className="shrink-0">
          <div
            className={[
              "inline-flex items-center gap-2 px-3 py-1 rounded-full border text-xs font-bold",
              severityBadgeClasses(severity),
            ].join(" ")}
          >
            <span>{severityLabel(severity)}</span>
          </div>
        </div>
      </div>

      {/* Painel de Telemetria (único conteúdo "gráfico") */}
      <div className="mt-4">
        <div className="mb-2 flex items-center justify-end gap-2 text-xs text-slate-300">
          <span className="text-slate-400">Intervalo:</span>
          <select
            value={windowCfg.value}
            onChange={(e) => setWindowKey(e.target.value)}
            className="bg-slate-950/70 border border-slate-700 rounded px-2 py-1"
          >
            {WINDOW_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </div>
        <TelemetryDashboard vm={telemetryVM} />
      </div>

      {/* Top services dentro do Host Overview */}
      <div className="mt-4">
        <TopServicesCard tenantId={tenantId} hostname={hostname} />
      </div>
    </div>
  );
}
