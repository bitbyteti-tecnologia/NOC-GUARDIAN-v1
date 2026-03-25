// CreateTenant.jsx
// - Cria um novo cliente (tenant) via POST /api/v1/tenants.
// - Após criar, mostra ações:
//    1) Abrir dashboard do cliente (/tenant/:id)
//    2) Voltar ao dashboard global com destaque (/?highlight=:id)
//
// RBAC:
// - Apenas superadmin/support (backend valida; aqui é visual).

import React, { useMemo, useState } from "react";
import api from "../lib/api";
import useMe from "../hooks/useMe";
import { Link } from "react-router-dom";

export default function CreateTenant() {
  const { me } = useMe();
  const [name, setName] = useState("");
  const [ipsRaw, setIpsRaw] = useState("");
  const [snmpVersion, setSnmpVersion] = useState("v2c");
  const [useSNMP, setUseSNMP] = useState(false);
  const [snmpCommunity, setSnmpCommunity] = useState("");
  const [snmpUser, setSnmpUser] = useState("");
  const [snmpAuthProto, setSnmpAuthProto] = useState("sha");
  const [snmpAuthPass, setSnmpAuthPass] = useState("");
  const [snmpPrivProto, setSnmpPrivProto] = useState("aes");
  const [snmpPrivPass, setSnmpPrivPass] = useState("");
  const [msg, setMsg] = useState("");
  const [created, setCreated] = useState(null); // {id, name, db_name}
  const [creating, setCreating] = useState(false);

  const ips = useMemo(() => {
    return String(ipsRaw || "")
      .split(/[\n,;\s]+/g)
      .map((v) => v.trim())
      .filter(Boolean);
  }, [ipsRaw]);
  const can = me && (me.role === "superadmin" || me.role === "support");
  if (!can) return <div className="text-slate-400">Sem permissão.</div>;

  function buildSNMP() {
    if (!useSNMP) return null;
    if (snmpVersion === "v3") {
      if (!snmpUser) return null;
      return {
        version: "v3",
        username: snmpUser,
        auth_protocol: snmpAuthProto || "",
        auth_password: snmpAuthPass || "",
        priv_protocol: snmpPrivProto || "",
        priv_password: snmpPrivPass || "",
      };
    }
    if (!snmpCommunity) return null;
    return {
      version: "v2c",
      community: snmpCommunity,
    };
  }

  async function onCreate(e, runDiscovery = false) {
    e.preventDefault();
    setMsg("");
    setCreated(null);
    setCreating(true);

    try {
      const payload = {
        name,
        ips,
        snmp: buildSNMP(),
      };
      const r = await api.post("/api/v1/tenants", payload);
      setCreated(r.data);
      if (runDiscovery) {
        await api.post(`/api/v1/tenants/${r.data.id}/discovery`, payload);
        setMsg("Cliente criado e discovery iniciado.");
      } else {
        setMsg("Cliente criado com sucesso.");
      }
    } catch (e) {
      setMsg("Falha ao criar cliente. Verifique logs/back-end.");
    } finally {
      setCreating(false);
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Criar novo cliente</h1>

      <form onSubmit={(e) => onCreate(e, false)} className="card space-y-4 max-w-2xl">
        <div>
          <label className="text-sm">Nome do Cliente</label>
          <input
            className="w-full p-2 rounded text-slate-900"
            value={name}
            onChange={e=>setName(e.target.value)}
            placeholder="Cliente Matriz"
          />
        </div>

        <div>
          <label className="text-sm">IPs da rede</label>
          <textarea
            className="w-full p-2 rounded text-slate-900 min-h-[96px]"
            value={ipsRaw}
            onChange={(e) => setIpsRaw(e.target.value)}
            placeholder="Ex: 10.0.0.1, 10.0.0.2 ou um IP por linha"
          />
          <div className="text-xs text-slate-500 mt-1">
            {ips.length} IP(s) detectado(s).
          </div>
        </div>

        <div className="rounded-lg border border-slate-800 bg-slate-950/60 p-3">
          <div className="flex items-center justify-between gap-3">
            <div className="text-sm font-semibold text-slate-100">Credencial SNMP (opcional)</div>
            <label className="flex items-center gap-2 text-xs text-slate-300">
              <input
                type="checkbox"
                checked={useSNMP}
                onChange={(e) => setUseSNMP(e.target.checked)}
              />
              Informar credenciais
            </label>
          </div>
          {useSNMP && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mt-2">
            <div>
              <label className="text-xs text-slate-400">Versão</label>
              <select
                className="w-full p-2 rounded text-slate-900"
                value={snmpVersion}
                onChange={(e) => setSnmpVersion(e.target.value)}
              >
                <option value="v2c">v2c</option>
                <option value="v3">v3</option>
              </select>
            </div>

            {snmpVersion === "v2c" && (
              <div>
                <label className="text-xs text-slate-400">Community</label>
                <input
                  className="w-full p-2 rounded text-slate-900"
                  value={snmpCommunity}
                  onChange={(e) => setSnmpCommunity(e.target.value)}
                  placeholder="public"
                />
              </div>
            )}

            {snmpVersion === "v3" && (
              <>
                <div>
                  <label className="text-xs text-slate-400">Username</label>
                  <input
                    className="w-full p-2 rounded text-slate-900"
                    value={snmpUser}
                    onChange={(e) => setSnmpUser(e.target.value)}
                    placeholder="snmpuser"
                  />
                </div>
                <div>
                  <label className="text-xs text-slate-400">Auth Protocol</label>
                  <select
                    className="w-full p-2 rounded text-slate-900"
                    value={snmpAuthProto}
                    onChange={(e) => setSnmpAuthProto(e.target.value)}
                  >
                    <option value="sha">SHA</option>
                    <option value="sha256">SHA-256</option>
                    <option value="md5">MD5</option>
                  </select>
                </div>
                <div>
                  <label className="text-xs text-slate-400">Auth Password</label>
                  <input
                    className="w-full p-2 rounded text-slate-900"
                    type="password"
                    value={snmpAuthPass}
                    onChange={(e) => setSnmpAuthPass(e.target.value)}
                  />
                </div>
                <div>
                  <label className="text-xs text-slate-400">Priv Protocol</label>
                  <select
                    className="w-full p-2 rounded text-slate-900"
                    value={snmpPrivProto}
                    onChange={(e) => setSnmpPrivProto(e.target.value)}
                  >
                    <option value="aes">AES</option>
                    <option value="aes256">AES-256</option>
                    <option value="des">DES</option>
                  </select>
                </div>
                <div>
                  <label className="text-xs text-slate-400">Priv Password</label>
                  <input
                    className="w-full p-2 rounded text-slate-900"
                    type="password"
                    value={snmpPrivPass}
                    onChange={(e) => setSnmpPrivPass(e.target.value)}
                  />
                </div>
              </>
            )}
          </div>
          )}
          {!useSNMP && (
            <div className="text-xs text-slate-500 mt-2">
              Se não informar credenciais, o discovery pode rodar sem SNMP (seed de devices).
            </div>
          )}
        </div>

        <div className="flex flex-col md:flex-row gap-2">
          <button className="px-4 py-2 bg-sky-600 rounded hover:bg-sky-500" disabled={creating}>
            {creating ? "Criando..." : "Criar"}
          </button>
          <button
            type="button"
            className="px-4 py-2 bg-emerald-700 rounded hover:bg-emerald-600"
            disabled={creating}
            onClick={(e) => onCreate(e, true)}
          >
            Descobrir rede automaticamente
          </button>
        </div>

        {msg && <div className="text-sm text-slate-300">{msg}</div>}
      </form>

      {created && (
        <div className="card space-y-3">
          <div className="font-semibold">Cliente criado</div>
          <div className="text-sm text-slate-400">
            <div><b>ID:</b> {created.id}</div>
            <div><b>Nome:</b> {created.name}</div>
            <div><b>DB:</b> {created.db_name}</div>
          </div>

          <div className="flex flex-col md:flex-row gap-3">
            <Link
              to={`/tenant/${created.id}`}
              className="px-4 py-2 rounded bg-emerald-700 hover:bg-emerald-600 text-center"
            >
              Abrir dashboard do cliente
            </Link>

            <Link
              to={`/?highlight=${created.id}`}
              className="px-4 py-2 rounded bg-slate-800 hover:bg-slate-700 text-center"
            >
              Voltar ao dashboard global (destacar card)
            </Link>
          </div>
        </div>
      )}

      <div className="text-xs text-slate-500">
        Ao criar, a Central cria um DB isolado do tenant automaticamente e aplica migrações TimescaleDB.
      </div>
    </div>
  );
}
