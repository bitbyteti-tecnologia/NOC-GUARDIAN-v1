import React from "react";
import { clampPct, formatPct, formatBytes } from "../format";

function ProgressBar({ value, label, sublabel, colorClass, loadInfo }) {
  const pct = clampPct(value) ?? 0;
  // Estilo htop: [|||||      20%]
  const bars = Math.floor(pct / 4); // 25 barras total
  
  // Cores dinâmicas por traço (verde -> amarelo -> vermelho)
  const renderBars = () => {
    const segments = [];
    for (let i = 0; i < 25; i++) {
      let charColor = "text-slate-700"; // fundo
      if (i < bars) {
        if (i < 15) charColor = "text-emerald-500"; // < 60%
        else if (i < 21) charColor = "text-amber-500"; // < 84%
        else charColor = "text-rose-500"; // > 84%
      }
      segments.push(<span key={i} className={charColor}>|</span>);
    }
    return segments;
  };

  return (
    <div className="flex flex-col space-y-0.5 font-mono text-[11px]">
      <div className="flex justify-between items-end px-1">
        <div className="flex items-center gap-2">
          <span className="text-slate-400 font-bold uppercase w-6">{label}</span>
          {loadInfo && <span className="text-slate-500 text-[10px]">{loadInfo}</span>}
        </div>
        <div className="flex items-center gap-2">
          {sublabel && <span className="text-slate-500 text-[10px]">{sublabel}</span>}
          <span className={`${colorClass} font-bold`}>{pct.toFixed(1)}%</span>
        </div>
      </div>
      <div className="relative h-5 bg-slate-950 border border-slate-800 rounded flex items-center px-2 overflow-hidden shadow-inner">
        <span className="relative z-10 whitespace-pre flex tracking-[-0.1em]">
          <span className="text-slate-500 mr-1">[</span>
          {renderBars()}
          <span className="text-slate-500 ml-1">]</span>
        </span>
      </div>
    </div>
  );
}

export function ResourceUsageCard({ data, host }) {
  const memUsed = data?.memUsedBytes;
  const memTotal = data?.memTotalBytes;

  const cpuCurrent = clampPct(data?.cpuPct);
  const memCurrent = clampPct(data?.memPct);
  const diskCurrent = clampPct(data?.diskPct);

  // Load Average do host (se disponível nas métricas em tempo real)
  // Tentando mapear tanto load1/5/15 quanto load_avg_1/5/15
  const l1 = host?.load_avg_1 ?? host?.load1;
  const l5 = host?.load_avg_5 ?? host?.load5;
  const l15 = host?.load_avg_15 ?? host?.load15;

  const loadStr = l1 != null 
    ? `${l1.toFixed(2)} ${l5?.toFixed(2)} ${l15?.toFixed(2)}`
    : null;

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-3 bg-slate-900/60 p-3 rounded-xl border border-white/5">
      {/* CPU */}
      <ProgressBar 
        label="CPU" 
        value={cpuCurrent} 
        colorClass="text-sky-400"
        loadInfo={loadStr ? `Load: ${loadStr}` : null}
      />

      {/* Memória */}
      <ProgressBar 
        label="MEM" 
        value={memCurrent} 
        sublabel={memUsed != null && memTotal != null ? `${formatBytes(memUsed)}/${formatBytes(memTotal)}` : null}
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