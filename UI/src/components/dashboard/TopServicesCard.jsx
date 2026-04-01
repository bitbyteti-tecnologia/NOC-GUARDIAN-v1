import React, { useEffect, useMemo, useState } from "react";
import api from "../../lib/api";

function formatServiceName(metric, prefix) {
  const rest = metric.replace(prefix, "");
  const parts = rest.split("_").filter(Boolean);
  if (parts.length >= 2 && /^\d+$/.test(parts[0])) {
    parts.shift();
  }
  const name = parts.join(" ");
  return name || metric;
}

function formatValue(metric, value) {
  if (metric.endsWith("_pct")) {
    return `${value.toFixed(1)}%`;
  }
  if (metric.endsWith("_bytes")) {
    const gb = value / (1024 * 1024 * 1024);
    return `${gb.toFixed(2)} GB`;
  }
  return value.toFixed(2);
}

function useLatestMetrics({ tenantId, hostname, prefix }) {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (!tenantId || !hostname || !prefix) return;
    let alive = true;
    setLoading(true);
    setError(false);

    api
      .get(`/api/v1/tenants/${tenantId}/dashboard/latest`, {
        params: {
          hostname,
          metric_prefix: prefix,
          limit: 20,
        },
      })
      .then((r) => {
        if (!alive) return;
        setItems(r.data?.items || []);
      })
      .catch(() => {
        if (!alive) return;
        setItems([]);
        setError(true);
      })
      .finally(() => {
        if (!alive) return;
        setLoading(false);
      });

    return () => {
      alive = false;
    };
  }, [tenantId, hostname, prefix]);

  return { items, loading, error };
}

export default function TopServicesCard({ tenantId, hostname }) {
  const cpuPrefix = "service_cpu_top";
  const memPrefix = "service_mem_top";
  const { items: cpuItems, loading: cpuLoading, error: cpuError } = useLatestMetrics({
    tenantId,
    hostname,
    prefix: cpuPrefix,
  });
  const { items: memItems, loading: memLoading, error: memError } = useLatestMetrics({
    tenantId,
    hostname,
    prefix: memPrefix,
  });

  const cpuTop = useMemo(() => cpuItems.slice(0, 5), [cpuItems]);
  const memTop = useMemo(() => memItems.slice(0, 5), [memItems]);

  const loading = cpuLoading || memLoading;
  const error = cpuError || memError;

  return (
    <div className="rounded-xl border border-slate-800 bg-slate-950/50 p-4">
      <div className="flex items-start justify-between gap-3 mb-4">
        <div>
          <div className="font-semibold text-slate-100">Top Services</div>
          <div className="text-xs text-slate-400 mt-1">
            Host: <span className="text-slate-200 font-semibold">{hostname || "-"}</span>
          </div>
        </div>
        {loading && <div className="text-xs text-slate-500">Carregando...</div>}
      </div>

      {error && (
        <div className="text-xs text-amber-300 mb-3">
          Não foi possível carregar os serviços agora.
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="rounded-lg border border-slate-800 bg-slate-950/60 p-3">
          <div className="text-xs text-slate-400 uppercase tracking-wide">CPU (Top 5)</div>
          <div className="mt-2 space-y-2">
            {cpuTop.length === 0 && <div className="text-xs text-slate-500">Sem dados.</div>}
            {cpuTop.map((item) => (
              <div key={item.metric} className="flex items-center justify-between text-sm">
                <span className="text-slate-200">{formatServiceName(item.metric, cpuPrefix)}</span>
                <span className="text-sky-300 font-semibold">{formatValue(item.metric, item.value)}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="rounded-lg border border-slate-800 bg-slate-950/60 p-3">
          <div className="text-xs text-slate-400 uppercase tracking-wide">Memória (Top 5)</div>
          <div className="mt-2 space-y-2">
            {memTop.length === 0 && <div className="text-xs text-slate-500">Sem dados.</div>}
            {memTop.map((item) => (
              <div key={item.metric} className="flex items-center justify-between text-sm">
                <span className="text-slate-200">{formatServiceName(item.metric, memPrefix)}</span>
                <span className="text-emerald-300 font-semibold">{formatValue(item.metric, item.value)}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

