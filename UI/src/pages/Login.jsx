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
    return (
      <div className="min-h-screen flex flex-col items-center justify-center bg-slate-950 text-slate-100">
        <div className="mb-8 animate-pulse">
          <img 
            src="/logo.png" 
            alt="NOC Guardian" 
            className="h-32 object-contain"
            onError={(e) => { e.target.src = "https://img.freepik.com/free-vector/magnifying-glass-with-world-globe_1308-129715.jpg"; }} 
          />
        </div>
        <div className="text-center space-y-4">
          <h2 className="text-2xl font-bold tracking-widest text-sky-400">INICIALIZANDO SISTEMA</h2>
          <div className="w-64 h-2 bg-slate-800 rounded-full overflow-hidden border border-slate-700">
            <div 
              className="h-full bg-sky-500 transition-all duration-1000 ease-linear"
              style={{ width: `${(10 - countdown) * 10}%` }}
            />
          </div>
          <p className="text-slate-400 font-mono text-sm">Carregando módulos de segurança... {countdown}s</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-slate-950 text-slate-100 p-4">
      <div className="mb-8">
        <img 
          src="/logo.png" 
          alt="NOC Guardian" 
          className="h-24 object-contain"
          onError={(e) => { e.target.style.display = 'none'; }}
        />
      </div>
      
      <form onSubmit={onSubmit} className="card w-full max-w-sm space-y-4 border border-white/5 bg-slate-900/50 backdrop-blur-xl">
        <div className="text-center mb-6">
          <h1 className="text-2xl font-black tracking-tight text-white uppercase italic">NOC Guardian</h1>
          <p className="text-xs text-slate-500 font-medium">MONITOR - PROTECT - OPTIMIZE</p>
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