import React, { useEffect, useMemo, useState } from "react";
import api from "../lib/api";

function pick(latest, name) {
  const it = Array.isArray(latest) ? latest.find(x => x.metric === name) : null;
  if (!it) return null;
  const n = Number(it.value);
  return Number.isNaN(n) ? null : n;
}

function fmtUptime(sec) {
  if (sec === null || sec === undefined) return "—";
  const s = Math.max(0, Math.floor(sec));
  const days = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  const ss = s % 60;

  const pad = (x) => String(x).padStart(2, "0");
  const core = `${pad(h)}:${pad(m)}:${pad(ss)}`;
  return days > 0 ? `${days} days, ${core}` : core;
}

function fmtLoad(a, b, c) {
  const f = (x) => (x === null || x === undefined) ? "—" : x.toFixed(2);
  return `${f(a)}, ${f(b)}, ${f(c)}`;
}

export default function CpuExtraInfo({ tenantID, deviceID }) {
  const [latest, setLatest] = useState([]);
  const [loading, setLoading] = useState(false);

  const metrics = useMemo(() => (
    "uptime_sec,load1,load5,load15,proc_count,thread_count,kthread_count,running_procs"
  ), []);

  useEffect(() => {
    if (!tenantID || !deviceID) return;

    setLoading(true);
    const url =
      `/api/v1/${tenantID}/metrics/latest` +
      `?device_id=${encodeURIComponent(deviceID)}` +
      `&metric=${encodeURIComponent(metrics)}`;

    api.get(url)
      .then(r => setLatest(Array.isArray(r.data) ? r.data : []))
      .catch(() => setLatest([]))
      .finally(() => setLoading(false));
  }, [tenantID, deviceID, metrics]);

  const uptime = pick(latest, "uptime_sec");
  const load1 = pick(latest, "load1");
  const load5 = pick(latest, "load5");
  const load15 = pick(latest, "load15");

  const proc = pick(latest, "proc_count");
  const thr = pick(latest, "thread_count");
  const kthr = pick(latest, "kthread_count");
  const run = pick(latest, "running_procs");

  // Serviços
  const services = [
    { name: "Docker", val: pick(latest, "service_docker_status") },
    { name: "Nginx", val: pick(latest, "service_nginx_status") },
    { name: "Postgres", val: pick(latest, "service_postgresql_status") },
    { name: "Central", val: pick(latest, "service_central_status") },
    { name: "Dash", val: pick(latest, "service_dashboard_status") },
  ];

  const hasLoad = (load1 !== null || load5 !== null || load15 !== null);

  return (
    <div className="mt-2 text-[11px] leading-4 text-slate-300/90 font-mono">
      <div className="flex flex-col gap-y-2">
        {/* Tasks e Uptime */}
        <div className="flex flex-wrap gap-x-4">
          <div>
            <span className="text-slate-500 font-bold uppercase">Tasks:</span>{" "}
            <span>{proc === null ? "—" : Math.round(proc)}</span>,{" "}
            <span>{thr === null ? "—" : Math.round(thr)}</span> thr,{" "}
            <span>{run === null ? "—" : Math.round(run)}</span> running
          </div>
          <div>
            <span className="text-slate-500 font-bold uppercase">Uptime:</span>{" "}
            <span>{fmtUptime(uptime)}</span>
          </div>
        </div>

        {/* Serviços */}
        <div className="flex flex-wrap gap-x-3 border-t border-white/5 pt-2">
          {services.map(s => s.val !== null && (
            <div key={s.name} className="flex items-center gap-1">
              <span className="text-slate-500 uppercase font-bold">{s.name}:</span>
              <span className={s.val === 1 ? "text-emerald-400" : "text-rose-500"}>
                {s.val === 1 ? "active" : "inactive"}
              </span>
            </div>
          ))}
        </div>
      </div>

      {loading && (
        <div className="mt-1 text-[10px] text-slate-500 italic">Atualizando...</div>
      )}
    </div>
  );
}
