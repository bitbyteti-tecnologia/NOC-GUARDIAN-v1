// health.js
// Pequenas funções utilitárias para calcular "saúde" do host
// a partir de CPU, memória, disco e status.

function toNumber(v) {
  const n = Number(v);
  return Number.isFinite(n) ? n : null;
}

export function computeHostSeverity(host) {
  const cpu = toNumber(host?.cpu_percent);
  const mem = toNumber(host?.mem_used_pct);
  const disk = toNumber(host?.disk_used_pct);

  const over90 = [cpu, mem, disk].some((v) => v != null && v >= 90);
  const over80 = [cpu, mem, disk].some((v) => v != null && v >= 80);

  if (host?.status === "OFFLINE") return "critical";
  if (over90) return "critical";
  if (over80) return "warning";
  return "ok";
}

export function severityLabel(sev) {
  if (sev === "critical") return "CRÍTICO";
  if (sev === "warning") return "ATENÇÃO";
  return "OK";
}

export function severityBadgeClasses(sev) {
  if (sev === "critical") {
    return "bg-red-500/15 text-red-200 border-red-500/40";
  }
  if (sev === "warning") {
    return "bg-amber-500/15 text-amber-200 border-amber-500/40";
  }
  return "bg-emerald-500/15 text-emerald-200 border-emerald-500/30";
}

export function buildHostHealthSummary(host) {
  const sev = computeHostSeverity(host);
  const cpu = toNumber(host?.cpu_percent);
  const mem = toNumber(host?.mem_used_pct);
  const disk = toNumber(host?.disk_used_pct);

  const parts = [];

  if (sev === "critical") {
    parts.push("Risco alto");
  } else if (sev === "warning") {
    parts.push("Risco moderado");
  } else {
    parts.push("Saúde geral boa");
  }

  if (cpu != null && cpu >= 80) {
    parts.push(`CPU elevada (${cpu.toFixed(1)}%)`);
  }
  if (mem != null && mem >= 80) {
    parts.push(`Memória alta (${mem.toFixed(1)}%)`);
  }
  if (disk != null && disk >= 90) {
    parts.push(`Disco quase cheio (${disk.toFixed(1)}%)`);
  }

  if (parts.length === 1 && sev === "ok") {
    return "Sem anomalias relevantes nos últimos minutos.";
  }

  return parts.join(" · ");
}

