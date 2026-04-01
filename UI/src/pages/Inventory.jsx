import React, { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import api from "../lib/api";

export default function Inventory() {
  const { tenantID } = useParams();
  const [hosts, setHosts] = useState([]);

  useEffect(() => {
    if (!tenantID) return;
    api
      .get(`/api/v1/tenants/${tenantID}/dashboard/hosts`)
      .then((r) => setHosts(r.data?.hosts || r.data || []))
      .catch(() => setHosts([]));
  }, [tenantID]);

  const rows = useMemo(() => {
    const arr = Array.isArray(hosts) ? [...hosts] : [];
    arr.sort((a, b) => (a.hostname || "").localeCompare(b.hostname || ""));
    return arr;
  }, [hosts]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Inventário</h1>
        <div className="text-xs text-slate-400 mt-1">
          Tenant: <span className="text-slate-200 font-mono">{tenantID}</span>
        </div>
      </div>

      <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
        <div className="font-semibold text-slate-100">Ativos monitorados</div>
        <div className="text-xs text-slate-400 mt-2">
          Hostname, sistema operacional e última leitura de telemetria.
        </div>
        <div className="mt-4 overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="text-left text-slate-400">
                <th className="py-2 pr-4">Host</th>
                <th className="py-2 pr-4">SO</th>
                <th className="py-2 pr-4">Status</th>
                <th className="py-2 pr-4">CPU</th>
                <th className="py-2 pr-4">Mem</th>
                <th className="py-2 pr-4">Disco</th>
                <th className="py-2 pr-4">Último contato</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((h) => (
                <tr key={h.hostname} className="border-t border-slate-800 text-slate-200">
                  <td className="py-2 pr-4">{h.hostname}</td>
                  <td className="py-2 pr-4">{h.os || "-"}</td>
                  <td className="py-2 pr-4 capitalize">{h.status || "-"}</td>
                  <td className="py-2 pr-4">{h.cpu_percent != null ? `${h.cpu_percent.toFixed(1)}%` : "-"}</td>
                  <td className="py-2 pr-4">{h.mem_used_pct != null ? `${h.mem_used_pct.toFixed(1)}%` : "-"}</td>
                  <td className="py-2 pr-4">{h.disk_used_pct != null ? `${h.disk_used_pct.toFixed(1)}%` : "-"}</td>
                  <td className="py-2 pr-4">
                    {h.last_seen ? new Date(h.last_seen).toLocaleString("pt-BR") : "-"}
                  </td>
                </tr>
              ))}
              {rows.length === 0 && (
                <tr>
                  <td className="py-4 text-slate-500" colSpan={7}>
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
