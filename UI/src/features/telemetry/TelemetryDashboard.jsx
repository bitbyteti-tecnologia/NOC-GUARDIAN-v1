import { useMemo, useState } from "react";
import { mapTelemetry } from "./adapters";
import { ResourceUsageCard } from "./components/ResourceUsageCard";
import { NetworkUsageCard } from "./components/NetworkUsageCard";
import { DiskIoCard } from "./components/DiskIoCard";
import { TelemetryStatusCard } from "./components/TelemetryStatusCard";

export function TelemetryDashboard({ raw, fieldMap, vm }) {
  const [range, setRange] = useState("1h");

  const computed = useMemo(() => {
    if (vm) return vm;
    if (raw && fieldMap) return mapTelemetry(raw, fieldMap);
    return null;
  }, [raw, fieldMap, vm]);

  if (!computed) {
    return (
      <div className="rounded-2xl bg-slate-900/60 p-4 shadow-lg ring-1 ring-white/10">
        <div className="text-sm text-slate-300">
          Telemetria indisponível: forneça <code className="text-slate-100">vm</code> ou{" "}
          <code className="text-slate-100">raw + fieldMap</code>.
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="rounded-2xl bg-slate-900/60 p-4 shadow-lg ring-1 ring-white/10">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <div className="text-sm font-semibold text-slate-100">
              Detalhes do Host: <span className="text-sky-300">{computed.host?.name}</span>
            </div>
            <div className="mt-1 text-xs text-slate-300">
              {computed.host?.ip ? <>IP: {computed.host.ip}</> : null}
              {computed.host?.os ? <>{" "} | OS: {computed.host.os}</> : null}
              {computed.host?.uptime ? <>{" "} | Uptime: {computed.host.uptime}</> : null}
            </div>
          </div>

          <div className="flex items-center gap-2">
            <span className="text-xs text-slate-400">Intervalo:</span>
            <select
              value={range}
              onChange={(e) => setRange(e.target.value)}
              className="rounded-xl bg-slate-950/50 px-3 py-2 text-sm text-slate-100 ring-1 ring-white/10 focus:outline-none"
            >
              <option value="15m">15m</option>
              <option value="30m">30m</option>
              <option value="1h">1h</option>
              <option value="6h">6h</option>
              <option value="24h">24h</option>
            </select>
          </div>
        </div>
      </div>

      <ResourceUsageCard data={computed.resources} />
      <NetworkUsageCard current={computed.network?.current} series={computed.network?.series} />
      <DiskIoCard current={computed.diskIO?.current} series={computed.diskIO?.series} />
      <TelemetryStatusCard flags={computed.flags} alerts={computed.alerts} />
    </div>
  );
}