import React, { useEffect, useState } from "react";
import api from "../lib/api";

export default function Sessions() {
  const [sessions, setSessions] = useState([]);
  const [msg, setMsg] = useState("");

  async function load() {
    try {
      const r = await api.get("/api/v1/auth/sessions");
      setSessions(r.data || []);
    } catch {
      setSessions([]);
    }
  }

  useEffect(() => { load(); }, []);

  async function revokeAll() {
    setMsg("");
    try {
      await api.post("/api/v1/auth/sessions/revoke-all");
      setMsg("Sessões revogadas. Redirecionando para login...");
      setTimeout(() => window.location.href = "/login", 1200);
    } catch {
      setMsg("Falha ao revogar sessões.");
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Sessões</h1>

      <div className="card flex items-center justify-between">
        <div>
          <div className="font-semibold">Controle de Sessões</div>
          <div className="text-sm text-slate-400">Revoga todas as sessões do usuário (logout global).</div>
        </div>
        <button onClick={revokeAll} className="px-4 py-2 bg-rose-600 rounded hover:bg-rose-500">
          Revogar tudo
        </button>
      </div>

      {msg && <div className="text-sm text-slate-300">{msg}</div>}

      <div className="card">
        <div className="font-semibold mb-2">Últimas sessões (até 50)</div>
        <table className="w-full text-sm">
          <thead className="text-slate-400">
            <tr>
              <th className="text-left">Criada</th>
              <th className="text-left">Expira</th>
              <th className="text-left">IP</th>
              <th className="text-left">User-Agent</th>
              <th className="text-left">Revogada</th>
            </tr>
          </thead>
          <tbody>
            {sessions.map(s => (
              <tr key={s.id} className="border-t border-slate-800">
                <td>{new Date(s.created_at).toLocaleString()}</td>
                <td>{new Date(s.expires_at).toLocaleString()}</td>
                <td>{s.ip || "-"}</td>
                <td className="truncate max-w-[420px]">{s.user_agent || "-"}</td>
                <td>{s.revoked_at ? new Date(s.revoked_at).toLocaleString() : "-"}</td>
              </tr>
            ))}
            {sessions.length === 0 && (
              <tr><td className="text-slate-500 py-3" colSpan="5">Nenhuma sessão encontrada.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}