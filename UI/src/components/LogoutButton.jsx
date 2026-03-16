import React from "react";
import api, { setAuthHeader } from "../lib/api";

export default function LogoutButton() {
  const logout = async () => {
    try { await api.post("/api/v1/auth/logout"); } catch {}
    localStorage.removeItem("token");
    setAuthHeader(null);
    window.location.href = "/login";
  };
  return (
    <button onClick={logout} className="px-3 py-1 bg-rose-600 rounded hover:bg-rose-500">
      Sair
    </button>
  );
}