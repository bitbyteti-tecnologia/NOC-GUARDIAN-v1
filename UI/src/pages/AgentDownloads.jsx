import React, { useMemo, useState } from "react";
import { useParams } from "react-router-dom";

export default function AgentDownloads() {
  const params = useParams();
  const tenantId = params.tenantId || params.tenantID || "";
  const token = tenantId || "TOKEN-NAO-DEFINIDO";
  const [copied, setCopied] = useState(false);

  const downloads = useMemo(
    () => [
      { label: "Windows (MSI x64)", file: "nocguardian-agent-windows-x64.msi", desc: "Windows Server 2012+ e Windows 10/11." },
      { label: "Linux AMD64 (.deb)", file: "nocguardian-agent_amd64.deb", desc: "Ubuntu/Debian amd64." },
      { label: "Linux ARM64 (.deb)", file: "nocguardian-agent_arm64.deb", desc: "Ubuntu/Debian arm64." },
      { label: "Linux x86_64 (.rpm)", file: "nocguardian-agent_x86_64.rpm", desc: "CentOS/RHEL x86_64." },
      { label: "Linux aarch64 (.rpm)", file: "nocguardian-agent_aarch64.rpm", desc: "CentOS/RHEL arm64." },
    ],
    []
  );

  async function copyToken() {
    try {
      await navigator.clipboard.writeText(token);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      setCopied(false);
    }
  }

  return (
    <div className="flex flex-col lg:flex-row gap-6">
      <aside className="w-full lg:w-64 bg-slate-950/80 border border-slate-800 rounded-xl p-4 h-fit">
        <div className="text-slate-100 font-semibold mb-4">NOC Panel</div>
        <nav className="space-y-2 text-sm">
          <div className="text-xs uppercase tracking-wider text-slate-500 mt-3">Monitoramento</div>
          <div className="text-slate-300">Dashboard</div>
          <div className="text-slate-300">Alertas</div>
          <div className="text-slate-300">Mapa de Topologia</div>

          <div className="text-xs uppercase tracking-wider text-slate-500 mt-4">Análise</div>
          <div className="text-slate-300">Relatórios</div>
          <div className="text-slate-300">Inventário</div>

          <div className="text-xs uppercase tracking-wider text-slate-500 mt-4">Gestão</div>
          <div className="text-slate-300">Configurações</div>
          <div className="text-slate-100 font-semibold">Downloads & Instalação</div>
        </nav>
      </aside>

      <div className="flex-1 space-y-6">
        <div>
          <h1 className="text-2xl font-bold">Download de Agentes</h1>
          <div className="text-xs text-slate-400 mt-1">
            Use o token do tenant para ativar os agentes.
          </div>
        </div>

        <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4 flex flex-col md:flex-row md:items-center md:justify-between gap-3">
          <div className="text-sm">
            Seu Token de Ativação: <span className="font-mono text-slate-100">{token}</span>
          </div>
          <button
            className="px-3 py-2 bg-sky-600 rounded text-sm font-semibold hover:bg-sky-500"
            onClick={copyToken}
          >
            {copied ? "Copiado" : "Copiar"}
          </button>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {downloads.map((d) => (
            <div key={d.file} className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
              <div className="text-lg font-semibold">{d.label}</div>
              <div className="text-xs text-slate-400 mt-1">{d.desc}</div>
              <a
                href={`/downloads/${d.file}`}
                className="inline-block mt-4 px-4 py-2 bg-sky-600 rounded text-sm font-semibold hover:bg-sky-500"
                download
              >
                Baixar
              </a>
            </div>
          ))}

          <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
            <div className="text-lg font-semibold">Docker</div>
            <div className="text-xs text-slate-400 mt-1">
              Ideal para monitorar containers e microserviços.
            </div>
            <code className="block bg-slate-900/80 text-slate-100 text-xs p-3 mt-3 rounded">
              docker pull nocguardian/agent:latest
            </code>
          </div>

          <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
            <div className="text-lg font-semibold">SNMP / Network</div>
            <div className="text-xs text-slate-400 mt-1">
              Use o discovery para varrer ativos SNMP e iniciar a topologia.
            </div>
            <code className="block bg-slate-900/80 text-slate-100 text-xs p-3 mt-3 rounded">
              discovery: /tenant/{tenantId || "{tenantId}"} /discovery
            </code>
          </div>
        </div>
      </div>
    </div>
  );
}

