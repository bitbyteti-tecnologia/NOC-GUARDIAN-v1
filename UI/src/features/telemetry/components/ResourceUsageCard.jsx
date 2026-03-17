import React from "react";
import { clampPct, formatPct, formatBytes } from "../format";

function ProgressBar({ value, label, sublabel, colorClass }) {
  const pct = clampPct(value);
  // Estilo htop: [|||||      20%]
  const bars = Math.floor(pct / 4); // 25 barras total
  const barStr = "|".repeat(bars) + " ".repeat(25 - bars);

  return (
    <div className="flex flex-col space-y-1 font-mono text-[11px]">
      <div className="flex justify-between items-end">
        <span className="text-slate-400 font-bold uppercase tracking-tighter">{label}</span>
        <div className="flex flex-col items-end">
          <span className={`${colorClass} font-bold`}>{pct.toFixed(1)}%</span>
          {sublabel && <span className="text-slate-500 text-[10px]">{sublabel}</span>}
        </div>
      </div>
      <div className="relative h-4 bg-slate-900/80 border border-slate-700/50 rounded flex items-center px-2 overflow-hidden">
        <div 
          className={`absolute left-0 top-0 bottom-0 ${colorClass.replace('text-', 'bg-')}/30 transition-all duration-500`} 
          style={{ width: `${pct}%` }}
        />
        <span className="relative z-10 text-slate-300 whitespace-pre">
          [{barStr}]
        </span>
      </div>
    </div>
  );
}

export function ResourceUsageCard({ data }) {
  const memUsed = data?.memUsedBytes;
  const memTotal = data?.memTotalBytes;

  const cpuCurrent = clampPct(data?.cpuPct);
  const memCurrent = clampPct(data?.memPct);
  const diskCurrent = clampPct(data?.diskPct);

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-4 bg-slate-900/40 p-4 rounded-2xl border border-white/5 shadow-inner">
      {/* CPU */}
      <ProgressBar 
        label="CPU" 
        value={cpuCurrent} 
        colorClass="text-sky-400"
      />

      {/* Memória */}
      <ProgressBar 
        label="MEM" 
        value={memCurrent} 
        sublabel={memUsed != null && memTotal != null ? `${formatBytes(memUsed)} / ${formatBytes(memTotal)}` : null}
        colorClass="text-fuchsia-400"
      />

      {/* Disco */}
      <ProgressBar 
        label="DSK" 
        value={diskCurrent} 
        colorClass="text-emerald-400"
      />
    </div>
  );
}