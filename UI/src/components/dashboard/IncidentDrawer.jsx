import React, { useMemo, useState } from "react";
import ChartLine from "../ChartLine";

function fmtDuration(from) {
  if (!from) return "-";
  const start = new Date(from).getTime();
  if (!Number.isFinite(start)) return "-";
  const diff = Date.now() - start;
  const min = Math.floor(diff / 60000);
  if (min < 60) return `${min} min`;
  const h = Math.floor(min / 60);
  const m = min % 60;
  return `${h}h ${m}m`;
}

function Skeleton() {
  return (
    <div className="animate-pulse space-y-3">
      <div className="h-5 w-40 bg-slate-800/80 rounded" />
      <div className="h-4 w-64 bg-slate-800/80 rounded" />
      <div className="h-4 w-52 bg-slate-800/80 rounded" />
      <div className="h-32 w-full bg-slate-800/80 rounded" />
    </div>
  );
}

export default function IncidentDrawer({ open, onClose, loading, error, data }) {
  const [metricKey, setMetricKey] = useState("");

  const metrics = data?.metrics || [];
  const activeMetric = useMemo(() => {
    if (!metrics.length) return null;
    if (metricKey) return metrics.find((m) => m.metric === metricKey) || metrics[0];
    return metrics[0];
  }, [metrics, metricKey]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="absolute right-0 top-0 h-full w-full max-w-3xl bg-slate-950 border-l border-slate-800 shadow-2xl overflow-y-auto">
        <div className="p-5 border-b border-slate-800 flex items-center justify-between">
          <div>
            <div className="text-sm text-slate-400">Incidente</div>
            <div className="text-lg font-semibold text-slate-100">{data?.incident?.root_event || "Detalhes"}</div>
          </div>
          <button
            className="px-3 py-2 bg-slate-900 border border-slate-700 rounded hover:bg-slate-800 text-xs"
            onClick={onClose}
          >
            Fechar
          </button>
        </div>

        <div className="p-5 space-y-6">
          {loading ? (
            <Skeleton />
          ) : error ? (
            <div className="text-sm text-rose-200">Falha ao carregar detalhes do incidente.</div>
          ) : (
            <>
              {/* Detalhes */}
              <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
                <div className="text-sm font-semibold text-slate-100">Detalhes</div>
                <div className="mt-2 grid grid-cols-1 md:grid-cols-2 gap-3 text-sm text-slate-300">
                  <div>ID: <span className="text-slate-100">{data?.incident?.incident_id}</span></div>
                  <div>Severidade: <span className="text-slate-100">{data?.incident?.severity}</span></div>
                  <div>Impacto: <span className="text-slate-100">{data?.incident?.impact_count}</span></div>
                  <div>Duração: <span className="text-slate-100">{fmtDuration(data?.incident?.created_at)}</span></div>
                </div>
              </div>

              {/* Timeline */}
              <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
                <div className="text-sm font-semibold text-slate-100">Timeline</div>
                <div className="mt-3 space-y-2 text-sm text-slate-300">
                  {(data?.timeline || []).length === 0 ? (
                    <div className="text-slate-400">Sem eventos de timeline.</div>
                  ) : (
                    data.timeline.map((t, idx) => (
                      <div key={idx} className="flex items-center justify-between">
                        <div>{t.type} • {t.event_type}</div>
                        <div className="text-xs text-slate-500">{new Date(t.ts).toLocaleString()}</div>
                      </div>
                    ))
                  )}
                </div>
              </div>

              {/* Devices */}
              <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
                <div className="text-sm font-semibold text-slate-100">Devices afetados</div>
                <div className="mt-3 space-y-2 text-sm text-slate-300">
                  {(data?.devices || []).length === 0 ? (
                    <div className="text-slate-400">Nenhum device listado.</div>
                  ) : (
                    data.devices.map((d) => (
                      <div key={d.device_id} className="flex items-center justify-between">
                        <div className="font-mono text-xs text-slate-300">{d.device_id}</div>
                        <div className="text-xs text-slate-400">{d.status}</div>
                      </div>
                    ))
                  )}
                </div>
              </div>

              {/* Métricas */}
              <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
                <div className="flex items-center justify-between">
                  <div className="text-sm font-semibold text-slate-100">Métricas recentes</div>
                  <select
                    className="bg-slate-950/70 border border-slate-700 rounded px-2 py-1 text-xs"
                    value={activeMetric?.metric || ""}
                    onChange={(e) => setMetricKey(e.target.value)}
                  >
                    {metrics.map((m) => (
                      <option key={m.metric} value={m.metric}>{m.metric}</option>
                    ))}
                  </select>
                </div>
                <div className="mt-3">
                  {activeMetric ? (
                    <ChartLine data={activeMetric.points || []} />
                  ) : (
                    <div className="text-slate-400 text-sm">Sem métricas recentes.</div>
                  )}
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
