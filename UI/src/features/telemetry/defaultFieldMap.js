export const defaultFieldMap = {
  hostName: "host.name",
  hostIp: "host.ip",
  hostOs: "host.os",
  hostUptime: "host.uptime",

  cpuPct: "metrics.cpu.pct",
  memPct: "metrics.mem.pct",
  diskPct: "metrics.disk.pct",
  diskMount: "metrics.disk.mount",

  netCurrentRxBps: "net.current.rxBps",
  netCurrentTxBps: "net.current.txBps",
  netSeries: "net.series",
  netSeriesTs: "ts",
  netSeriesRxBps: "rxBps",
  netSeriesTxBps: "txBps",

  diskReadBps: "disk.current.readBps",
  diskWriteBps: "disk.current.writeBps",
  diskSeries: "disk.series",
  diskSeriesTs: "ts",
  diskSeriesReadBps: "readBps",
  diskSeriesWriteBps: "writeBps",

  flagNetOk: "flags.telemetry.net",
  flagDiskOk: "flags.telemetry.disk",

  alerts: "alerts.lastHour",
  alertTs: "ts",
  alertMessage: "message",
  alertSeverity: "severity",
};