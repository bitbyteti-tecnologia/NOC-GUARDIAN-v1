// api.js
// - Axios instance para chamadas da UI ao backend.
// - Mantém Authorization com token do localStorage.
// - Interceptor: em 401 tenta POST /api/v1/auth/refresh (cookie HttpOnly) e repete a request.

import axios from "axios";

const api = axios.create({});

export function setAuthHeader(token) {
  if (token) api.defaults.headers.common["Authorization"] = `Bearer ${token}`;
  else delete api.defaults.headers.common["Authorization"];
}

// aplica token salvo
const saved = localStorage.getItem("token");
if (saved) setAuthHeader(saved);

let refreshing = false;
let queue = [];

api.interceptors.response.use(
  (resp) => resp,
  async (error) => {
    const original = error.config || {};
    const status = error.response?.status;

    if (status === 401 && !original._retry) {
      original._retry = true;

      try {
        if (!refreshing) {
          refreshing = true;
          const r = await axios.post("/api/v1/auth/refresh");
          const newToken = r.data?.token;

          if (newToken) {
            localStorage.setItem("token", newToken);
            setAuthHeader(newToken);
          }

          refreshing = false;
          queue.forEach(fn => fn(newToken));
          queue = [];
        } else {
          await new Promise(resolve => queue.push(resolve));
        }

        return api(original);
      } catch (e) {
        refreshing = false;
        queue = [];
        localStorage.removeItem("token");
        setAuthHeader(null);
        if (!window.location.pathname.startsWith("/login")) {
          window.location.href = "/login";
        }
      }
    }

    return Promise.reject(error);
  }
);

export default api;