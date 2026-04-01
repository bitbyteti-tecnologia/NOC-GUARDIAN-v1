import React, { useMemo, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import useMe from "../hooks/useMe";
import useSessionAge from "../hooks/useSessionAge";
import LogoutButton from "./LogoutButton";

function NavItem({ to, label, onClick, disabled }) {
  const { pathname } = useLocation();
  const active = pathname === to;
  if (disabled) {
    return (
      <div className="block px-3 py-2 rounded-lg text-sm text-slate-500 cursor-not-allowed">
        {label}
      </div>
    );
  }
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
  const isTenantView = Boolean(tenantId);

  const [open, setOpen] = useState(false);
  const [cfgOpen, setCfgOpen] = useState(false);

  const closeAll = () => { setOpen(false); setCfgOpen(false); };

  return (
    <div className="sticky top-0 z-40 bg-[#020617] backdrop-blur">
      <div className="relative px-4 md:px-6 h-[200px] flex items-center justify-between overflow-visible">
        {/* subtle pattern */}
        <div
          className="pointer-events-none absolute inset-0 opacity-10"
          style={{ backgroundImage: "radial-gradient(circle, #475569 1px, transparent 1px)", backgroundSize: "20px 20px" }}
        />
        {/* top line */}
        <div className="pointer-events-none absolute top-0 w-full h-px bg-gradient-to-r from-transparent via-slate-500 to-transparent" />

        <div className="flex items-center gap-3 w-56 relative z-10">
          <button
            className="px-3 py-2 rounded hover:bg-slate-900 border border-slate-800"
            onClick={() => setOpen(!open)}
            aria-label="Menu"
            title="Menu"
          >
            ☰
          </button>
        </div>

        <div className="absolute left-1/2 -translate-x-1/2 z-10 w-full max-w-5xl px-4">
          <Link to="/" className="flex items-center justify-center w-full">
            <div className="flex items-center justify-center w-full max-w-5xl h-full px-4 relative">
              <div className="flex-1 h-[2px] bg-gradient-to-r from-transparent to-slate-500 opacity-50" />
              <div className="relative flex items-center justify-center px-10 h-full">
                <img
                  src="/LogoNOCGuardian.png"
                  alt="NOC Guardian Logo"
                  className="relative z-10 h-[130px] w-auto object-contain drop-shadow-[0_0_18px_rgba(59,130,246,0.75)] transition-transform duration-300 group-hover:scale-105"
                  onError={(e) => { e.target.style.display = "none"; }}
                />
              </div>
              <div className="flex-1 h-[2px] bg-gradient-to-l from-transparent to-slate-500 opacity-50" />
            </div>
          </Link>
        </div>

        <div className="flex items-center gap-3 relative z-10">
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

        {/* central flare removed */}
      </div>

      {/* Menu lateral (mobile + desktop) */}
      {open && (
        <>
          <div className="fixed inset-0 bg-black/50 z-50" onClick={closeAll} />
          <aside
            className="fixed left-0 top-[200px] h-[calc(100vh-200px)] w-80 max-w-[85vw] bg-slate-950 border-r border-slate-800 z-[60] flex flex-col"
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
              {!isTenantView && (
                <>
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
                </>
              )}

              {isTenantView && (
                <>
                  <div className="text-xs uppercase tracking-wider text-slate-500">Monitoramento</div>
                  <NavItem to={`/tenant/${tenantId}`} label="Dashboard" onClick={closeAll} />
                  <NavItem to={`/tenant/${tenantId}/alerts`} label="Alertas Ativos" onClick={closeAll} />
                  <NavItem to={`/tenant/${tenantId}#topologia`} label="Mapa de Topologia" onClick={closeAll} />

                  <div className="text-xs uppercase tracking-wider text-slate-500 pt-3">Análise</div>
                  <NavItem to="#" label="Relatórios Executivo" disabled />
                  <NavItem to="#" label="SLA & Disponibilidade" disabled />
                  <NavItem to="#" label="KPIs de Performance" disabled />
                  <NavItem to="#" label="Consumo & Billing" disabled />
                  <NavItem to="#" label="Inventário" disabled />

                  <div className="text-xs uppercase tracking-wider text-slate-500 pt-3">Gestão</div>
                  {isTenantOperator && tenantId && (
                    <NavItem to={`/tenant/${tenantId}/users`} label="Gestão de Acessos" onClick={closeAll} />
                  )}
                  <NavItem to={`/tenant/${tenantId}/downloads`} label="Downloads & Instalação" onClick={closeAll} />
                  <NavItem to={`/tenant/${tenantId}#discovery`} label="Descoberta (Scan)" onClick={closeAll} />

                  <div className="text-xs uppercase tracking-wider text-slate-500 pt-3">Suporte</div>
                  <NavItem to="#" label="Central de Alertas" disabled />
                  <NavItem to="#" label="Base de Conhecimento" disabled />
                  <NavItem to="#" label="Suporte & Tickets" disabled />
                </>
              )}
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
