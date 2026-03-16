export type Health = "OK" | "AVISO" | "CRÍTICO";

export type TimeRange = "15m" | "30m" | "1h" | "6h";

export interface HostDetails {
  name: string;
  ip: string;
  os: string;
  uptime: string;
}

export interface ResourceUsage {
  cpuPercent: number;      // 0-100
  memPercent: number;      // 0-100
  diskPercent: number;     // 0-100 (root "/")
}

export interface NetworkPoint {
  ts: number;              // epoch ms
  rxBps: number;           // bytes/s
  txBps: number;           // bytes/s
}

export interface DiskIo {
  readBps: number;         // bytes/s
  writeBps: number;        // bytes/s
}

export interface TelemetryFlags {
  netOk: boolean;
  diskOk: boolean;
}

export interface AlertItem {
  ts: number;              // epoch ms
  message: string;
  severity: Health;
}

export interface TelemetryPayload {
  host: HostDetails;
  resources: ResourceUsage;
  networkHistory: NetworkPoint[];
  currentRxBps: number;
  currentTxBps: number;
  diskIo: DiskIo;
  flags: TelemetryFlags;
  alertHistory: AlertItem[];
}
