import { useMemo, useState } from "react";
import { TelemetryVM, TimeRange } from "./models";
import { TelemetryFieldMap } from "./fieldMap";
import { mapTelemetry } from "./adapters";
import { ResourceUsageCard } from "./components/ResourceUsageCard";
import { NetworkUsageCard } from "./components/NetworkUsageCard";
import { DiskIoCard } from "./components/DiskIoCard";
import { TelemetryStatusCard } from "./components/TelemetryStatusCard";

export function TelemetryDashboard({
  raw,
  fieldMap,
  vm: vmProp,
}: {
  raw?: any;                 // payload real do seu sistema
  fieldMap?: TelemetryFieldMap; // mapeamento de paths
  vm?: TelemetryVM;          // opcional: se você já adaptar fora
}) {
  const [range, setRange] = useState<TimeRange>("1h");

  const vm = useMemo(() => {
    if (vmProp) return vmProp;
    if (raw && fieldMap) return mapTelemetry(raw, fieldMap);
    return undefined;
  }, [raw, fieldMap, vmProp]);

  if (!vm) {
    return (
      <div className="rounded-2xl bg-slate-900/60 p-4 shadow-lg ring-1 ring-white/10">
        <div className="text-sm text-slate-300">
          Telemetria indisponível: forneça <code className="text-slate-100">vm</code> ou <code className="text-slate-100">raw + fieldMap</code>.
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header Host Details */}
      <div className="rounded-2xl bg-slate-900/60 p-4 shadow-lg ring-1 ring-white/10">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <div className="text-sm font-semibold text-slate-100">
              Detalhes do Host: <span className="text-sky-300">{vm.host.name}</span>
            </div>
            <div className="mt-1 text-xs text-slate-300">
              {vm.host.ip ? <>IP: {vm.host.ip}</> : null}
              {vm.host.os ? <>{" "} | OS: {vm.host.os}</> : null}
              {vm.host.uptime ? <>{" "} | Uptime: {vm.host.uptime}</> : null}
            </div>
          </div>

          <div className="flex items-center gap-2">
            <span className="text-xs text-slate-400">Intervalo:</span>
            <select
              value={range}
              onChange={(e) => setRange(e.target.value as TimeRange)}
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

      {/* Cards */}
      <ResourceUsageCard data={vm.resources} />
      <NetworkUsageCard current={vm.network.current} series={vm.network.series} />
      <DiskIoCard current={vm.diskIO.current} series={vm.diskIO.series} />
      <TelemetryStatusCard flags={vm.flags} alerts={vm.alerts} />
    </div>
  );
}
