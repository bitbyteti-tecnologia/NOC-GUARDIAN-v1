import React, { useMemo, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import useMe from "../hooks/useMe";
import useSessionAge from "../hooks/useSessionAge";
import LogoutButton from "./LogoutButton";

function NavItem({ to, label, onClick }) {
  const { pathname } = useLocation();
  const active = pathname === to;
  return (
    <Link
      to={to}
      onClick={onClick}
      className={`block px-3 py-2 rounded-lg text-sm ${active ? "bg-slate-800" : "hover:bg-slate-900"}`}
    >
      {label}
    </Link>
  );
}

export default function Topbar() {
  const { me } = useMe();
  const sessionAge = useSessionAge();
  const isGlobalAdmin = me && (me.role === "superadmin" || me.role === "support");
  const { pathname } = useLocation();
  const tenantId = useMemo(() => {
    const m = String(pathname || "").match(/^\/tenant\/([^/]+)/);
    return m ? m[1] : "";
  }, [pathname]);
  const isTenantOperator = me && me.role === "admin" && me.tenant_id === tenantId;

  const [open, setOpen] = useState(false);
  const [cfgOpen, setCfgOpen] = useState(false);

  const closeAll = () => { setOpen(false); setCfgOpen(false); };

  return (
    <div className="sticky top-0 z-50 border-b border-slate-800 bg-slate-950/90 backdrop-blur">
      <div className="px-4 md:px-6 py-3 flex items-center justify-between">

        <div className="flex items-center gap-3">
          <button
            className="px-3 py-2 rounded hover:bg-slate-900 border border-slate-800"
            onClick={() => setOpen(!open)}
            aria-label="Menu"
            title="Menu"
          >
            ☰
          </button>

          <Link to="/" className="flex items-center gap-3 group">
            <img 
              src="/Logo NOC - Guardian-01-Transparente.png" 
              alt="Logo" 
              className="h-8 w-auto object-contain transition-transform group-hover:scale-105"
              onError={(e) => { e.target.style.display = 'none'; }}
            />
            <div className="font-bold text-lg hidden sm:block tracking-tight text-white italic">NOC Guardian</div>
          </Link>

          {/* Desktop nav removido (menu lateral unificado) */}
        </div>

        <div className="flex items-center gap-3">
          <div className="hidden md:block text-xs text-slate-300">
            Tempo: <span className="text-slate-100">{sessionAge}</span>
          </div>

          {me && (
            <div className="hidden md:block text-xs text-slate-300">
              {me.email} <span className="text-slate-500">({me.role})</span>
            </div>
          )}
          <LogoutButton />
        </div>
      </div>

      {/* Menu lateral (mobile + desktop) */}
      {open && (
        <>
          <div className="fixed inset-0 bg-black/50 z-40" onClick={closeAll} />
          <aside className="fixed inset-y-0 left-0 w-80 max-w-[85vw] bg-slate-950 border-r border-slate-800 z-50 flex flex-col">
            <div className="flex items-center justify-between px-4 py-3 border-b border-slate-800">
              <div className="text-sm font-semibold tracking-wide text-slate-200">Menu</div>
              <button
                className="px-2 py-1 rounded hover:bg-slate-900 border border-slate-800"
                onClick={closeAll}
              >
                ✕
              </button>
            </div>

            <div className="flex-1 overflow-y-auto px-4 py-4 space-y-3">
              <div className="text-xs uppercase tracking-wider text-slate-500">Navegação</div>
              <NavItem to="/" label="Dashboard" onClick={closeAll} />

              <div className="text-xs uppercase tracking-wider text-slate-500 pt-3">Configurações</div>
              <NavItem to="/sessions" label="Sessões" onClick={closeAll} />
              {isGlobalAdmin && <NavItem to="/users" label="Usuários Globais" onClick={closeAll} />}
              {isGlobalAdmin && <NavItem to="/create-tenant" label="Criar novo cliente" onClick={closeAll} />}
              {isTenantOperator && tenantId && (
                <NavItem to={`/tenant/${tenantId}/users`} label="Usuários do Cliente" onClick={closeAll} />
              )}
              <NavItem to="/change-password" label="Alterar senha" onClick={closeAll} />

              <div className="text-xs uppercase tracking-wider text-slate-500 pt-3">Downloads de agentes</div>
              <a className="block px-3 py-2 rounded-lg text-sm hover:bg-slate-900" href="/downloads/nocguardian-agent-windows-x64.msi" onClick={closeAll}>
                Windows (MSI)
              </a>
              <a className="block px-3 py-2 rounded-lg text-sm hover:bg-slate-900" href="/downloads/nocguardian-agent_amd64.deb" onClick={closeAll}>
                Linux AMD64 (.deb)
              </a>
              <a className="block px-3 py-2 rounded-lg text-sm hover:bg-slate-900" href="/downloads/nocguardian-agent_arm64.deb" onClick={closeAll}>
                Linux ARM64 (.deb)
              </a>
              <a className="block px-3 py-2 rounded-lg text-sm hover:bg-slate-900" href="/downloads/nocguardian-agent_x86_64.rpm" onClick={closeAll}>
                Linux x86_64 (rpm)
              </a>
              <a className="block px-3 py-2 rounded-lg text-sm hover:bg-slate-900" href="/downloads/nocguardian-agent_aarch64.rpm" onClick={closeAll}>
                Linux ARM64 (rpm)
              </a>
            </div>

            <div className="px-4 py-3 border-t border-slate-800 text-xs text-slate-300">
              Tempo: <span className="text-slate-100">{sessionAge}</span>
            </div>
          </aside>
        </>
      )}
    </div>
  );
}
