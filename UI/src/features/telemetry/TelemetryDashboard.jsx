import { useMemo } from "react";
import { mapTelemetry } from "./adapters";
import { ResourceUsageCard } from "./components/ResourceUsageCard";
import CpuExtraInfo from "../../components/CpuExtraInfo";
import { NetworkUsageCard } from "./components/NetworkUsageCard";
import { DiskIoCard } from "./components/DiskIoCard";
import { TelemetryStatusCard } from "./components/TelemetryStatusCard";

export function TelemetryDashboard({ raw, fieldMap, vm }) {
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
    <div className="space-y-6 mt-2">
      <div className="flex flex-col lg:flex-row gap-6">
        <div className="flex-1">
          <ResourceUsageCard data={computed.resources} host={computed.host} />
        </div>
        <div className="lg:w-80">
          <CpuExtraInfo 
            tenantID={computed.host?.tenant_id} 
            deviceID={computed.host?.id} 
          />
        </div>
      </div>
      <NetworkUsageCard current={computed.network?.current} series={computed.network?.series} />
      <DiskIoCard current={computed.diskIO?.current} series={computed.diskIO?.series} />
      <TelemetryStatusCard flags={computed.flags} alerts={computed.alerts} />
    </div>
  );
}