import { clampPct, formatPct } from "../format";

function pctToColor(p?: number) {
  const v = clampPct(p);
  if (v == null) return "text-slate-400";
  if (v < 70) return "text-emerald-400";
  if (v < 90) return "text-amber-400";
  return "text-rose-400";
}

export function Gauge({ value, label }: { value?: number; label: string }) {
  const v = clampPct(value);
  const angle = v == null ? 0 : (v / 100) * 180; // 0..180
  const stroke = 14;

  // arco: semicirculo (via stroke-dasharray)
  const r = 54;
  const c = Math.PI * r;
  const filled = v == null ? 0 : (v / 100) * c;

  return (
    <div className="flex items-center gap-4">
      <div className="relative h-28 w-32">
        <svg viewBox="0 0 140 90" className="h-full w-full">
          <path
            d="M 20 80 A 50 50 0 0 1 120 80"
            fill="none"
            stroke="rgba(148,163,184,0.25)"
            strokeWidth={stroke}
            strokeLinecap="round"
          />
          <path
            d="M 20 80 A 50 50 0 0 1 120 80"
            fill="none"
            stroke="currentColor"
            className={pctToColor(v)}
            strokeWidth={stroke}
            strokeLinecap="round"
            strokeDasharray={`${filled} ${c}`}
          />
          {/* ponteiro */}
          <g transform={`translate(70 80) rotate(${180 - angle})`}>
            <line x1="0" y1="0" x2="0" y2="-45" stroke="rgba(226,232,240,0.9)" strokeWidth="3" />
            <circle cx="0" cy="0" r="5" fill="rgba(226,232,240,0.95)" />
          </g>
        </svg>

        <div className="absolute inset-x-0 bottom-1 text-center">
          <div className="text-xs text-slate-300">{label}</div>
          <div className={`text-lg font-semibold ${pctToColor(v)}`}>{formatPct(v)}</div>
        </div>
      </div>
    </div>
  );
}
