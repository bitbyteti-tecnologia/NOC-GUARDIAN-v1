import React, { useState } from "react";
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

  const [open, setOpen] = useState(false);
  const [cfgOpen, setCfgOpen] = useState(false);

  const closeAll = () => { setOpen(false); setCfgOpen(false); };

  return (
    <div className="sticky top-0 z-50 border-b border-slate-800 bg-slate-950/90 backdrop-blur">
      <div className="px-4 md:px-6 py-3 flex items-center justify-between">

        <div className="flex items-center gap-3">
          <button
            className="md:hidden px-3 py-2 rounded hover:bg-slate-900"
            onClick={() => setOpen(!open)}
            aria-label="Menu"
            title="Menu"
          >
            ☰
          </button>

          <div className="font-bold text-lg">NOC Guardian</div>

          {/* Desktop nav */}
          <div className="hidden md:flex items-center gap-2 ml-4">
            <Link to="/" className="px-3 py-2 rounded-lg text-sm hover:bg-slate-900">Dashboard</Link>

            {/* Config dropdown (desktop) */}
            <div className="relative">
              <button
                onClick={() => setCfgOpen(!cfgOpen)}
                className="px-3 py-2 rounded-lg text-sm hover:bg-slate-900"
              >
                Configurações ▾
              </button>
              {cfgOpen && (
                <div className="absolute mt-2 w-56 bg-slate-950 border border-slate-800 rounded-xl p-2 shadow-lg">
                  <NavItem to="/sessions" label="Sessões" onClick={closeAll} />
                  {isGlobalAdmin && <NavItem to="/users" label="Usuários Globais" onClick={closeAll} />}
                  {isGlobalAdmin && <NavItem to="/create-tenant" label="Criar novo cliente" onClick={closeAll} />}
                  <NavItem to="/change-password" label="Alterar senha" onClick={closeAll} />
                </div>
              )}
            </div>
          </div>
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

      {/* Mobile menu */}
      {open && (
        <div className="md:hidden px-4 pb-3 space-y-2">
          <NavItem to="/" label="Dashboard" onClick={closeAll} />
          <button
            onClick={() => setCfgOpen(!cfgOpen)}
            className="w-full text-left px-3 py-2 rounded-lg text-sm hover:bg-slate-900"
          >
            Configurações ▾
          </button>
          {cfgOpen && (
            <div className="pl-2 space-y-1">
              <NavItem to="/sessions" label="Sessões" onClick={closeAll} />
              {isGlobalAdmin && <NavItem to="/users" label="Usuários Globais" onClick={closeAll} />}
              {isGlobalAdmin && <NavItem to="/create-tenant" label="Criar novo cliente" onClick={closeAll} />}
              <NavItem to="/change-password" label="Alterar senha" onClick={closeAll} />
            </div>
          )}
          <div className="text-xs text-slate-300 pt-2">
            Tempo: <span className="text-slate-100">{sessionAge}</span>
          </div>
        </div>
      )}
    </div>
  );
}