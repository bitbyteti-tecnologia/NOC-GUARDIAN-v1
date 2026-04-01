import React, { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import api from "../lib/api";
import HealthCard from "../components/dashboard/HealthCard";
import IncidentsCard from "../components/dashboard/IncidentsCard";
import InsightsCard from "../components/dashboard/InsightsCard";
import RecommendationsCard from "../components/dashboard/RecommendationsCard";
import NeuralFocusCard from "../components/dashboard/NeuralFocusCard";
import IncidentDrawer from "../components/dashboard/IncidentDrawer";
import useMe from "../hooks/useMe";

export default function CustomerNeural() {
  const params = useParams();
  const navigate = useNavigate();
  const { me } = useMe();
  const tenantId = params.tenantId || params.id || params.tenantID || params.tenant || "";
  const isSuperAdmin = me?.role === "superadmin";

  const [tenantName, setTenantName] = useState("");
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);
  const [incidentOpen, setIncidentOpen] = useState(false);
  const [incidentLoading, setIncidentLoading] = useState(false);
  const [incidentError, setIncidentError] = useState(false);
  const [incidentDetails, setIncidentDetails] = useState(null);

  async function loadTenantInfo() {
    if (!tenantId) return;
    try {
      const r = await api.get(`/api/v1/tenants/${tenantId}`);
      setTenantName(r.data?.name || "");
    } catch {
      setTenantName("");
    }
  }

  async function loadNeural() {
    if (!tenantId) return;
    setLoading(true);
    setError(false);
    try {
      const r = await api.get(`/api/v1/dashboard/neural`, {
        headers: {
          "X-Tenant-Id": tenantId,
        },
      });
      setData(r.data || null);
    } catch {
      setData(null);
      setError(true);
    } finally {
      setLoading(false);
    }
  }

  async function openIncident(inc) {
    if (!inc?.incident_id || !tenantId) return;
    setIncidentOpen(true);
    setIncidentLoading(true);
    setIncidentError(false);
    try {
      const r = await api.get(`/api/v1/dashboard/incidents/${encodeURIComponent(inc.incident_id)}/details`, {
        headers: {
          "X-Tenant-Id": tenantId,
        },
      });
      setIncidentDetails(r.data || null);
    } catch {
      setIncidentDetails(null);
      setIncidentError(true);
    } finally {
      setIncidentLoading(false);
    }
  }

  useEffect(() => {
    loadTenantInfo();
    loadNeural();
    // eslint-disable-next-line
  }, [tenantId]);

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">Dashboard Neural</h1>
          <div className="text-xs text-slate-400 mt-1">
            Cliente: <span className="text-slate-200">{tenantName || "(sem nome)"}</span>
            {tenantId ? (
              <span className="ml-2 text-slate-500 font-mono">({tenantId})</span>
            ) : null}
          </div>
          <div className="mt-3 inline-flex rounded-full border border-slate-700 bg-slate-900/50 p-1 text-xs">
            <Link
              to={`/tenant/${tenantId}`}
              className="rounded-full px-3 py-1 text-slate-300 hover:text-slate-100"
            >
              Padrão
            </Link>
            <span className="rounded-full px-3 py-1 bg-sky-600 text-white font-semibold">
              Neural
            </span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {isSuperAdmin && (
            <button
              className="px-3 py-2 bg-slate-900 border border-slate-700 rounded hover:bg-slate-800 text-sm"
              onClick={() => navigate("/")}
            >
              Voltar
            </button>
          )}
          <button
            className="px-4 py-2 bg-sky-600 rounded hover:bg-sky-500 font-semibold"
            onClick={loadNeural}
            disabled={loading}
          >
            {loading ? "..." : "Atualizar"}
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
        <div className="xl:col-span-2">
          <HealthCard data={data?.intelligence} loading={loading} error={error} />
        </div>
        <NeuralFocusCard data={data} loading={loading} error={error} />
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
        <IncidentsCard
          items={data?.intelligence?.top_incidents || []}
          loading={loading}
          error={error}
          onSelect={openIncident}
        />
        <InsightsCard
          items={data?.intelligence?.insights || []}
          loading={loading}
          error={error}
        />
        <RecommendationsCard
          items={data?.intelligence?.recommendations || []}
          loading={loading}
          error={error}
        />
      </div>

      <IncidentDrawer
        open={incidentOpen}
        onClose={() => setIncidentOpen(false)}
        loading={incidentLoading}
        error={incidentError}
        data={incidentDetails}
      />
    </div>
  );
}
