import React, { useEffect, useMemo, useState } from "react";
import { TelemetryDashboard } from "../features/telemetry/TelemetryDashboard";
import { useTelemetryFromApi } from "../features/telemetry/integrations/useTelemetryFromApi";
import { buildHostHealthSummary, computeHostSeverity, severityBadgeClasses, severityLabel } from "../features/telemetry/health";

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
    window: "1h",
    enabled: open,
    pollMs: 15000,
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
        <TelemetryDashboard vm={telemetryVM} />
      </div>
    </div>
  );
}
