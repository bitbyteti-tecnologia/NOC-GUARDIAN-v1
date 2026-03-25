import React, { useMemo } from "react";
import ReactFlow, { Background, Controls, MiniMap } from "reactflow";
import dagre from "@dagrejs/dagre";
import "reactflow/dist/style.css";

const statusStyles = {
  critical: { border: "#ef4444", bg: "rgba(239, 68, 68, 0.12)", text: "#fecaca" },
  warning: { border: "#f59e0b", bg: "rgba(245, 158, 11, 0.14)", text: "#fde68a" },
  info: { border: "#38bdf8", bg: "rgba(56, 189, 248, 0.12)", text: "#bae6fd" },
  ok: { border: "#22c55e", bg: "rgba(34, 197, 94, 0.12)", text: "#bbf7d0" },
  unknown: { border: "#94a3b8", bg: "rgba(148, 163, 184, 0.12)", text: "#e2e8f0" },
};

function layoutNodes(nodes, edges) {
  if (!nodes.length) return [];
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: "LR", nodesep: 50, ranksep: 100, marginx: 20, marginy: 20 });

  nodes.forEach((n) => {
    g.setNode(n.id, { width: 180, height: 70 });
  });
  edges.forEach((e) => {
    g.setEdge(e.source, e.target);
  });

  dagre.layout(g);

  return nodes.map((n) => {
    const pos = g.node(n.id);
    if (!pos) return n;
    return {
      ...n,
      position: { x: pos.x - 90, y: pos.y - 35 },
    };
  });
}

function mapStatus(status) {
  const key = String(status || "unknown").toLowerCase();
  return statusStyles[key] || statusStyles.unknown;
}

function fmtPct(val) {
  if (typeof val !== "number" || Number.isNaN(val)) return "—";
  return `${val.toFixed(1)}%`;
}

function fmtDate(val) {
  if (!val) return "—";
  try {
    return new Intl.DateTimeFormat("pt-BR", {
      dateStyle: "short",
      timeStyle: "short",
      timeZone: "America/Sao_Paulo",
    }).format(new Date(val));
  } catch {
    return String(val);
  }
}

function DeviceNode({ data }) {
  const style = mapStatus(data?.status);
  const isRoot = Boolean(data?.root);
  const incidents = data?.incident_count ?? 0;

  return (
    <div className="group relative">
      <div
        style={{
          border: `2px solid ${isRoot ? "#f97316" : style.border}`,
          background: isRoot ? "rgba(249, 115, 22, 0.18)" : style.bg,
          color: style.text,
          borderRadius: 14,
          padding: 12,
          fontSize: 12,
          minWidth: 160,
          boxShadow: isRoot ? "0 0 0 2px rgba(249, 115, 22, 0.15)" : "0 0 0 1px rgba(148,163,184,0.12)",
        }}
      >
        <div className="text-sm font-semibold">{data?.label || data?.id}</div>
        <div className="text-[11px] text-slate-300 mt-1">
          {data?.status || "unknown"} • incidentes: {incidents}
        </div>
      </div>

      <div className="pointer-events-none absolute left-1/2 top-full z-10 hidden w-56 -translate-x-1/2 translate-y-2 rounded-xl border border-slate-700 bg-slate-950/95 p-3 text-[11px] text-slate-200 shadow-xl group-hover:block">
        <div className="font-semibold text-slate-100 mb-1">Detalhes</div>
        <div className="space-y-1">
          <div>CPU: <span className="text-slate-100">{fmtPct(data?.metrics?.cpu_percent)}</span></div>
          <div>Memória: <span className="text-slate-100">{fmtPct(data?.metrics?.mem_used_pct)}</span></div>
          <div>Disco: <span className="text-slate-100">{fmtPct(data?.metrics?.disk_used_pct)}</span></div>
          <div>Último: <span className="text-slate-100">{fmtDate(data?.last_seen)}</span></div>
        </div>
      </div>
    </div>
  );
}

export default function TopologyCard({ data, loading, error }) {
  const { nodes, edges, roots } = useMemo(() => {
    const rawNodes = Array.isArray(data?.nodes) ? data.nodes : [];
    const rawEdges = Array.isArray(data?.edges) ? data.edges : [];
    const roots = rawNodes.filter((n) => n.root);

    const flowNodes = rawNodes.map((n) => ({
      id: n.id,
      type: "device",
      data: {
        id: n.id,
        label: n.label || n.id,
        status: n.status || "unknown",
        root: Boolean(n.root),
        incident_count: n.incident_count ?? 0,
        metrics: n.metrics || {},
        last_seen: n.last_seen,
      },
    }));

    const flowEdges = rawEdges.map((e, i) => ({
      id: `${e.source}-${e.target}-${i}`,
      source: e.source,
      target: e.target,
      label: e.relation_type || "",
      animated: false,
      style: { stroke: "rgba(148,163,184,0.45)", strokeWidth: 1.5 },
      labelStyle: { fill: "#94a3b8", fontSize: 10 },
    }));

    return {
      nodes: layoutNodes(flowNodes, flowEdges),
      edges: flowEdges,
      roots,
    };
  }, [data]);

  if (loading) {
    return (
      <div className="rounded-2xl bg-slate-900/60 p-4 shadow-lg ring-1 ring-white/10">
        <div className="h-64 w-full animate-pulse rounded-xl bg-slate-900/80" />
        <div className="mt-3 h-4 w-40 animate-pulse rounded bg-slate-900/80" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-2xl bg-slate-900/60 p-4 shadow-lg ring-1 ring-white/10">
        <div className="text-sm text-rose-200">Falha ao carregar topologia.</div>
      </div>
    );
  }

  if (!nodes.length) {
    return (
      <div className="rounded-2xl bg-slate-900/60 p-4 shadow-lg ring-1 ring-white/10">
        <div className="text-sm text-slate-300">Sem dados de topologia no momento.</div>
      </div>
    );
  }

  return (
    <div className="rounded-2xl bg-slate-900/60 p-4 shadow-lg ring-1 ring-white/10">
      <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
        <div>
          <div className="text-sm font-semibold tracking-wide text-slate-100">Topologia com Causa Raiz</div>
          <div className="text-xs text-slate-400">Dispositivos e dependências destacadas por severidade.</div>
        </div>
        <div className="flex flex-wrap items-center gap-2 text-[11px] text-slate-300">
          <span className="rounded-full border border-emerald-500/40 bg-emerald-500/10 px-2 py-1">OK</span>
          <span className="rounded-full border border-amber-500/40 bg-amber-500/10 px-2 py-1">Warning</span>
          <span className="rounded-full border border-rose-500/40 bg-rose-500/10 px-2 py-1">Critical</span>
          <span className="rounded-full border border-orange-500/40 bg-orange-500/10 px-2 py-1">Root cause</span>
        </div>
      </div>

      <div className="h-[420px] w-full overflow-hidden rounded-xl bg-slate-950/40 ring-1 ring-white/5">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={{ device: DeviceNode }}
          fitView
          fitViewOptions={{ padding: 0.2 }}
          nodesDraggable
          nodesConnectable={false}
          zoomOnScroll
          panOnScroll
        >
          <Background color="rgba(148,163,184,0.18)" gap={28} />
          <MiniMap
            pannable
            nodeStrokeColor={(n) => mapStatus(n?.data?.status).border}
            nodeColor={(n) => mapStatus(n?.data?.status).bg}
          />
          <Controls showInteractive={false} />
        </ReactFlow>
      </div>

      <div className="mt-3 text-xs text-slate-400">
        {roots.length > 0 ? (
          <span>
            Root cause identificado: {roots.map((r) => r.label || r.id).join(", ")}
          </span>
        ) : (
          <span>Root cause ainda não identificado. Relacionamentos adicionais ajudam a inferir upstream.</span>
        )}
      </div>
    </div>
  );
}
