import React, { useState, useEffect } from "react";
import api, { setAuthHeader } from "../lib/api";

export default function Login() {
  const [email, setEmail] = useState("bitbyteti@gmail.com");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);
  const [countdown, setCountDown] = useState(10);

  async function onSubmit(e) {
    e.preventDefault();
    setErr("");
    try {
      const resp = await api.post("/api/v1/auth/login", { email, password });
      const { token } = resp.data;
      localStorage.setItem("token", token);
      localStorage.setItem("login_at", Date.now().toString());
      setAuthHeader(token);
      
      // Inicia o processo de carregamento de 10 segundos
      setLoading(true);
    } catch (e) {
      setErr("Credenciais inválidas");
    }
  }

  useEffect(() => {
    if (!loading) return;
    
    if (countdown <= 0) {
      window.location.href = "/";
      return;
    }

    const timer = setTimeout(() => {
      setCountDown(countdown - 1);
    }, 1000);

    return () => clearTimeout(timer);
  }, [loading, countdown]);

  if (loading) {
    const statusMessages = [
      "Estabelecendo conexão segura com Data Center...",
      "Realizando handshake com API Gateway...",
      "Autenticando via túnel criptografado TLS 1.3...",
      "Sincronizando banco de dados TimescaleDB...",
      "Carregando módulos de telemetria em tempo real...",
      "Validando chaves de criptografia RSA-4096...",
      "Mapeando infraestrutura de rede global...",
      "Iniciando painel de controle NOC Guardian...",
      "Acesso autorizado. Carregando interface...",
      "Conexão estabelecida com sucesso!"
    ];
    const currentMessage = statusMessages[Math.min(9 - countdown, 9)];

    return (
      <div className="min-h-screen flex flex-col items-center justify-center bg-slate-950 text-slate-100 font-mono overflow-hidden">
        {/* Fundo de grade tecnológica sutil */}
        <div className="absolute inset-0 opacity-10 pointer-events-none" 
             style={{ backgroundImage: 'radial-gradient(#38bdf8 1px, transparent 1px)', backgroundSize: '30px 30px' }}></div>
        
        <div className="relative z-10 w-full max-w-lg px-6 flex flex-col items-center">
          <div className="mb-12 relative">
            <img 
              src="/LogoNOCGuardian.png" 
              alt="NOC Guardian" 
              className="h-32 object-contain relative z-10 drop-shadow-[0_0_15px_rgba(56,189,248,0.5)]"
              onError={(e) => { e.target.src = "/LogoNOCGuardian1.png"; }} 
            />
            {/* Círculo de pulso tecnológico atrás da logo */}
            <div className="absolute inset-0 bg-sky-500/20 rounded-full blur-3xl animate-pulse scale-150"></div>
          </div>

          <div className="w-full bg-slate-900/80 border border-sky-500/30 rounded-xl p-6 backdrop-blur-sm shadow-2xl shadow-sky-900/20">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-sm font-bold tracking-[0.2em] text-sky-400 uppercase">System Initialization</h2>
              <span className="text-xs text-sky-500/70">{100 - (countdown * 10)}%</span>
            </div>

            <div className="w-full h-1.5 bg-slate-800 rounded-full overflow-hidden mb-6 border border-slate-700/50">
              <div 
                className="h-full bg-sky-500 transition-all duration-1000 ease-linear shadow-[0_0_10px_#38bdf8]"
                style={{ width: `${(10 - countdown) * 10}%` }}
              />
            </div>

            <div className="space-y-2">
              <div className="flex items-center gap-3 text-sky-400/90 animate-pulse">
                <span className="w-2 h-2 bg-sky-500 rounded-full"></span>
                <p className="text-xs tracking-wide leading-relaxed">{currentMessage}</p>
              </div>
              
              {/* Log de status "hacker" */}
              <div className="mt-4 pt-4 border-t border-slate-800 space-y-1">
                <p className="text-[10px] text-slate-500 uppercase tracking-tighter">
                  {"\u203A "}IP: {Math.floor(Math.random()*255)}.{Math.floor(Math.random()*255)}.{Math.floor(Math.random()*255)}.{Math.floor(Math.random()*255)}
                </p>
                <p className="text-[10px] text-slate-500 uppercase tracking-tighter">
                  {"\u203A "}NODE: NOC-SVR-PRD-{Math.floor(Math.random()*999)}
                </p>
                <p className="text-[10px] text-emerald-500/70 uppercase tracking-tighter animate-bounce">
                  {"\u203A "}ENCRYPTION: ACTIVE
                </p>
              </div>
            </div>
          </div>

          <div className="mt-8 text-[10px] text-slate-600 font-bold tracking-[0.3em] uppercase">
            Secured by NOC Guardian Defense
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-slate-950 text-slate-100 p-4">
      <form onSubmit={onSubmit} className="card w-full max-w-sm space-y-4 border border-white/5 bg-slate-900/50 backdrop-blur-xl">
        <div className="flex flex-col items-center mb-6">
          <img 
            src="/LogoNOCGuardian.png" 
            alt="NOC Guardian" 
            className="h-28 w-auto object-contain mb-1"
            onError={(e) => { e.target.src = "/LogoNOCGuardian1.png"; }}
          />
          <div className="text-[10px] font-black tracking-[0.25em] text-sky-400/90 uppercase text-center mt-[-10px] drop-shadow-[0_0_8px_rgba(56,189,248,0.4)]">
            Monitor - Protect - Optimize
          </div>
        </div>

        {err && (
          <div className="bg-red-500/10 border border-red-500/20 text-red-400 p-3 rounded text-sm text-center">
            {err}
          </div>
        )}

        <div className="space-y-1">
          <label className="block text-[10px] font-bold text-slate-500 uppercase tracking-wider ml-1">E-mail</label>
          <input 
            className="w-full bg-slate-950 border border-slate-800 rounded-lg p-3 text-white focus:outline-none focus:border-sky-500 transition-colors" 
            placeholder="seu@email.com"
            value={email} 
            onChange={e=>setEmail(e.target.value)} 
          />
        </div>

        <div className="space-y-1">
          <label className="block text-[10px] font-bold text-slate-500 uppercase tracking-wider ml-1">Senha</label>
          <input 
            type="password" 
            className="w-full bg-slate-950 border border-slate-800 rounded-lg p-3 text-white focus:outline-none focus:border-sky-500 transition-colors" 
            placeholder="••••••••"
            value={password} 
            onChange={e=>setPassword(e.target.value)} 
          />
        </div>

        <button className="w-full bg-sky-600 hover:bg-sky-500 text-white font-bold py-3 rounded-lg shadow-lg shadow-sky-900/20 transition-all transform active:scale-[0.98]">
          ENTRAR NO SISTEMA
        </button>

        <div className="text-center pt-2">
          <a className="text-xs text-slate-500 hover:text-slate-300 transition-colors underline" href="/forgot-password">
            Esqueci minha senha
          </a>
        </div>
      </form>
    </div>
  );
}
