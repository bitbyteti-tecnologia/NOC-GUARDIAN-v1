export function Led({ ok, label }) {
  const color = ok ? "bg-emerald-500" : "bg-rose-500";
  const glow = ok ? "shadow-emerald-500/40" : "shadow-rose-500/40";
  return (
    <div className="flex items-center gap-2">
      <span className={`h-3 w-3 rounded-full ${color} shadow-lg ${glow}`} />
      <span className="text-sm text-slate-200">
        {label}:{" "}
        <span className={ok ? "text-emerald-300" : "text-rose-300"}>{ok ? "OK" : "NOK"}</span>
      </span>
    </div>
  );
}