// useMe.js
// - Hook para buscar o usuário logado via GET /api/v1/auth/me.
// - Usado na Topbar/RBAC visual e telas que dependem de role.
// - Depende do axios configurado em ../lib/api (Authorization e auto-refresh).

import { useEffect, useState } from "react";
import api from "../lib/api";

export default function useMe() {
  const [me, setMe] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.get("/api/v1/auth/me")
      .then(r => setMe(r.data))
      .catch(() => setMe(null))
      .finally(() => setLoading(false));
  }, []);

  return { me, loading };
}