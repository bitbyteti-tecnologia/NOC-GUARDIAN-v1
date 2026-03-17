export type TelemetryFieldMap = {
  hostName: string;
  hostIp?: string;
  hostOs?: string;
  hostUptime?: string;

  cpuPct?: string;
  memPct?: string;
  diskPct?: string;
  diskMount?: string;

  netCurrentRxBps?: string;
  netCurrentTxBps?: string;
  netSeries?: string;         // array
  netSeriesTs?: string;       // dentro do item
  netSeriesRxBps?: string;    // dentro do item
  netSeriesTxBps?: string;    // dentro do item

  diskReadBps?: string;
  diskWriteBps?: string;
  diskSeries?: string;        // array
  diskSeriesTs?: string;      // dentro do item
  diskSeriesReadBps?: string; // dentro do item
  diskSeriesWriteBps?: string;// dentro do item

  flagNetOk?: string;
  flagDiskOk?: string;

  alerts?: string;            // array
  alertTs?: string;           // dentro do item
  alertMessage?: string;      // dentro do item
  alertSeverity?: string;     // dentro do item
};

/**
 * Exemplo (AJUSTE para o seu payload real):
 * export const defaultFieldMap: TelemetryFieldMap = {
 *   hostName: "host.name",
 *   hostIp: "host.ip",
 *   hostOs: "host.os",
 *   hostUptime: "host.uptime",
 *   cpuPct: "metrics.cpu.pct",
 *   memPct: "metrics.mem.pct",
 *   diskPct: "metrics.disk.root.pct",
 *   diskMount: "metrics.disk.root.mount",
 *   netCurrentRxBps: "net.current.rxBps",
 *   netCurrentTxBps: "net.current.txBps",
 *   netSeries: "net.series",
 *   netSeriesTs: "ts",
 *   netSeriesRxBps: "rxBps",
 *   netSeriesTxBps: "txBps",
 *   diskReadBps: "disk.current.readBps",
 *   diskWriteBps: "disk.current.writeBps",
 *   diskSeries: "disk.series",
 *   diskSeriesTs: "ts",
 *   diskSeriesReadBps: "readBps",
 *   diskSeriesWriteBps: "writeBps",
 *   flagNetOk: "flags.telemetry.net",
 *   flagDiskOk: "flags.telemetry.disk",
 *   alerts: "alerts.lastHour",
 *   alertTs: "ts",
 *   alertMessage: "message",
 *   alertSeverity: "severity",
 * };
 */
