export type TimeRange = "15m" | "30m" | "1h" | "6h" | "24h";

export type HostDetails = {
  name: string;
  ip?: string;
  os?: string;
  uptime?: string;
};

export type ResourceUsage = {
  cpuPct?: number;     // 0..100
  memPct?: number;     // 0..100
  diskPct?: number;    // 0..100
  diskMount?: string;  // ex: "/"
};

export type NetworkPoint = {
  ts: number;          // epoch ms
  rxBps?: number;      // bytes/s
  txBps?: number;      // bytes/s
};

export type DiskIoPoint = {
  ts: number;          // epoch ms
  readBps?: number;    // bytes/s
  writeBps?: number;   // bytes/s
};

export type TelemetryFlags = {
  netOk?: boolean;
  diskOk?: boolean;
};

export type AlertItem = {
  ts: number;          // epoch ms
  message: string;
  severity: "info" | "warning" | "critical";
};

export type TelemetryVM = {
  host: HostDetails;
  resources: ResourceUsage;
  network: {
    current?: { rxBps?: number; txBps?: number; totalBps?: number };
    series?: NetworkPoint[];
  };
  diskIO: {
    current?: { readBps?: number; writeBps?: number };
    series?: DiskIoPoint[];
  };
  flags?: TelemetryFlags;
  alerts?: AlertItem[];
};
