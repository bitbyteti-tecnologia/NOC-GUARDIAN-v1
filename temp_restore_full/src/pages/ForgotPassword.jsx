import React, { useState } from "react";
import axios from "axios";

export default function ForgotPassword() {
  const [email, setEmail] = useState("");
  const [msg, setMsg] = useState("");

  async function onSubmit(e) {
    e.preventDefault();
    setMsg("");
    try {
      await axios.post("/api/v1/auth/forgot-password", { email });
      setMsg("Se o e-mail existir, enviaremos instruções (em DEV, veja logs da Central).");
    } catch {
      setMsg("Se o e-mail existir, enviaremos instruções (em DEV, veja logs da Central).");
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-950 text-slate-100">
      <form onSubmit={onSubmit} className="card w-full max-w-sm space-y-4">
        <h1 className="text-xl font-bold">Recuperar senha</h1>
        <div>
          <label className="block text-sm mb-1">E-mail</label>
          <input className="w-full rounded p-2 text-slate-900" value={email} onChange={e=>setEmail(e.target.value)} />
        </div>
        <button className="w-full bg-sky-600 py-2 rounded hover:bg-sky-500">Enviar</button>
        {msg && <div className="text-sm text-slate-300">{msg}</div>}
      </form>
    </div>
  );
}