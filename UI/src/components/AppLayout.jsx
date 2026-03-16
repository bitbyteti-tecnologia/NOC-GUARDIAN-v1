/**
 * AppLayout.jsx
 * - Layout padrão para páginas autenticadas.
 * - Renderiza Topbar (RBAC visual) + conteúdo.
 */
import React from "react";
import Topbar from "./Topbar";

export default function AppLayout({ children }) {
  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      <Topbar />
      <div className="p-6">{children}</div>
    </div>
  );
}