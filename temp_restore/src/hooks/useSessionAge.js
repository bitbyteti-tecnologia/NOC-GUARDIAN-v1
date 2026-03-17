// useSessionAge.js
// - Retorna string "Tempo conectado: HH:MM:SS" baseada em localStorage.login_at.

import { useEffect, useState } from "react";

function fmt(sec) {
  const h = String(Math.floor(sec / 3600)).padStart(2, "0");
  const m = String(Math.floor((sec % 3600) / 60)).padStart(2, "0");
  const s = String(sec % 60).padStart(2, "0");
  return `${h}:${m}:${s}`;
}

export default function useSessionAge() {
  const [text, setText] = useState("00:00:00");

  useEffect(() => {
    const start = parseInt(localStorage.getItem("login_at") || "0", 10);
    if (!start) return;

    const tick = () => {
      const diff = Math.max(0, Math.floor((Date.now() - start) / 1000));
      setText(fmt(diff));
    };

    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, []);

  return text;
}