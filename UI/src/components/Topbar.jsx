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
    <div className="sticky top-0 z-40 border-b border-slate-800 bg-gradient-to-b from-slate-950 via-slate-900/90 to-slate-950 backdrop-blur">
      <div className="relative px-4 md:px-6 h-[150px] flex items-center justify-between overflow-hidden">
        {/* top/bottom long lines */}
        <div className="pointer-events-none absolute left-0 right-0 top-2 h-px bg-gradient-to-r from-transparent via-slate-500/70 to-transparent" />
        <div className="pointer-events-none absolute left-0 right-0 bottom-2 h-px bg-gradient-to-r from-transparent via-slate-600/50 to-transparent" />

        {/* angled corners to mimic frame */}
        <div className="pointer-events-none absolute left-8 top-2 h-4 w-16 border-t border-l border-slate-500/60 skew-x-[-20deg]" />
        <div className="pointer-events-none absolute right-8 top-2 h-4 w-16 border-t border-r border-slate-500/60 skew-x-[20deg]" />
        <div className="pointer-events-none absolute left-8 bottom-2 h-4 w-16 border-b border-l border-slate-600/50 skew-x-[20deg]" />
        <div className="pointer-events-none absolute right-8 bottom-2 h-4 w-16 border-b border-r border-slate-600/50 skew-x-[-20deg]" />
        <div className="flex items-center gap-3 w-56">
          <button
            className="px-3 py-2 rounded hover:bg-slate-900 border border-slate-800"
            onClick={() => setOpen(!open)}
            aria-label="Menu"
            title="Menu"
          >
            ☰
          </button>
        </div>

        <Link to="/" className="flex items-center justify-center group flex-1">
          <div className="relative px-12 py-2">
            {/* central plate with angled corners */}
            <div
              className="absolute inset-0 border border-slate-500/60 bg-slate-950/80"
              style={{
                clipPath: "polygon(6% 0, 94% 0, 100% 50%, 94% 100%, 6% 100%, 0 50%)",
                boxShadow: "0 0 40px rgba(59,130,246,0.25)",
              }}
            />
            {/* glow line */}
            <div className="absolute left-1/2 top-1/2 h-[2px] w-[70%] -translate-x-1/2 -translate-y-1/2 bg-gradient-to-r from-transparent via-sky-400/80 to-transparent" />
            {/* subtle speckle band */}
            <div className="absolute left-1/2 top-1/2 h-[18px] w-[75%] -translate-x-1/2 -translate-y-1/2 opacity-20 bg-[radial-gradient(circle_at_20%_40%,rgba(148,163,184,0.6),transparent_40%),radial-gradient(circle_at_80%_60%,rgba(148,163,184,0.5),transparent_45%)]" />
            <img
              src="/Logo NOC - Guardian-01-Transparente.png"
              alt="Logo"
              className="relative h-[110px] w-auto object-contain transition-transform group-hover:scale-105"
              onError={(e) => { e.target.style.display = "none"; }}
            />
          </div>
        </Link>

        <div className="flex items-center gap-3">
          <div className="hidden md:block text-xs text-slate-300 font-mono tracking-wide">
            Tempo: <span className="text-slate-100">{sessionAge}</span>
          </div>

          {me && (
            <div className="hidden md:block text-xs text-slate-300 font-mono tracking-wide">
              {me.email} <span className="text-slate-500">({me.role})</span>
            </div>
          )}
          <LogoutButton />
        </div>
      </div>

      {/* Menu lateral (mobile + desktop) */}
      {open && (
        <>
          <div className="fixed inset-0 bg-black/50 z-50" onClick={closeAll} />
          <aside
            className="fixed left-0 top-[150px] h-[calc(100vh-150px)] w-80 max-w-[85vw] bg-slate-950 border-r border-slate-800 z-[60] flex flex-col"
            onClick={(e) => e.stopPropagation()}
          >
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

            </div>

            <div className="px-4 py-3 border-t border-slate-800 flex items-center justify-between text-xs text-slate-300">
              <div>
                Tempo: <span className="text-slate-100">{sessionAge}</span>
              </div>
              <LogoutButton />
            </div>
          </aside>
        </>
      )}
    </div>
  );
}
