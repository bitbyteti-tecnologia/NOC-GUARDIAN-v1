import React, { useMemo } from "react";
import ReactFlow, { Background, Controls, MiniMap } from "reactflow";
import "reactflow/dist/style.css";

const statusStyles = {
  critical: { border: "#ef4444", bg: "rgba(239, 68, 68, 0.12)", text: "#fecaca" },
  warning: { border: "#f59e0b", bg: "rgba(245, 158, 11, 0.14)", text: "#fde68a" },
  info: { border: "#38bdf8", bg: "rgba(56, 189, 248, 0.12)", text: "#bae6fd" },
  ok: { border: "#22c55e", bg: "rgba(34, 197, 94, 0.12)", text: "#bbf7d0" },
  unknown: { border: "#94a3b8", bg: "rgba(148, 163, 184, 0.12)", text: "#e2e8f0" },
};

function layoutNodes(nodes) {
  if (!nodes.length) return [];
  const cols = Math.ceil(Math.sqrt(nodes.length));
  const gapX = 230;
  const gapY = 140;
  return nodes.map((n, i) => {
    const col = i % cols;
    const row = Math.floor(i / cols);
    return {
      ...n,
      position: { x: col * gapX, y: row * gapY },
    };
  });
}

function mapStatus(status) {
  const key = String(status || "unknown").toLowerCase();
  return statusStyles[key] || statusStyles.unknown;
}

export default function TopologyCard({ data, loading, error }) {
  const { nodes, edges, roots } = useMemo(() => {
    const rawNodes = Array.isArray(data?.nodes) ? data.nodes : [];
    const rawEdges = Array.isArray(data?.edges) ? data.edges : [];
    const roots = rawNodes.filter((n) => n.root);

    const flowNodes = rawNodes.map((n) => {
      const style = mapStatus(n.status);
      const isRoot = Boolean(n.root);
      return {
        id: n.id,
        data: {
          label: n.label || n.id,
          status: n.status || "unknown",
          root: isRoot,
        },
        style: {
          border: `2px solid ${isRoot ? "#f97316" : style.border}`,
          background: isRoot ? "rgba(249, 115, 22, 0.18)" : style.bg,
          color: style.text,
          borderRadius: 14,
          padding: 12,
          fontSize: 12,
          minWidth: 150,
          boxShadow: isRoot ? "0 0 0 2px rgba(249, 115, 22, 0.15)" : "0 0 0 1px rgba(148,163,184,0.12)",
        },
      };
    });

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
      nodes: layoutNodes(flowNodes),
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
