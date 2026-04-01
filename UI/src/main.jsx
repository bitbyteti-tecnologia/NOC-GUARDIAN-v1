import React from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, RouterProvider } from "react-router-dom";

import "./styles.css";
import "./lib/api";
import "react-grid-layout/css/styles.css";
import "react-resizable/css/styles.css";

import ProtectedRoute from "./components/ProtectedRoute";
import AppLayout from "./components/AppLayout";

import Tenants from "./pages/Tenants";
import Customer from "./pages/Customer";
import CustomerExecutive from "./pages/CustomerExecutive";
import Alerts from "./pages/Alerts";
import Login from "./pages/Login";
import GlobalUsers from "./pages/GlobalUsers";
import TenantUsers from "./pages/TenantUsers";
import ChangePassword from "./pages/ChangePassword";
import ForgotPassword from "./pages/ForgotPassword";
import ResetPassword from "./pages/ResetPassword";
import Sessions from "./pages/Sessions";
import CreateTenant from "./pages/CreateTenant";
import TelemetryTest from "./pages/TelemetryTest";
import AgentDownloads from "./pages/AgentDownloads";
import Reports from "./pages/Reports";
import Inventory from "./pages/Inventory";
import Support from "./pages/Support";

const wrap = (page) => (
  <ProtectedRoute>
    <AppLayout>{page}</AppLayout>
  </ProtectedRoute>
);

const router = createBrowserRouter([
  // Públicas
  { path: "/login", element: <Login /> },
  { path: "/forgot-password", element: <ForgotPassword /> },
  { path: "/reset-password", element: <ResetPassword /> },

  // Protegidas
  { path: "/", element: wrap(<Tenants />) },
  { path: "/users", element: wrap(<GlobalUsers />) },
  { path: "/sessions", element: wrap(<Sessions />) },
  { path: "/change-password", element: wrap(<ChangePassword />) },
  { path: "/create-tenant", element: wrap(<CreateTenant />) },

  { path: "/telemetry-test", element: wrap(<TelemetryTest />) },

  { path: "/tenant/:tenantID", element: wrap(<Customer />) },
  { path: "/tenant/:tenantID/executive", element: wrap(<CustomerExecutive />) },
  { path: "/tenant/:tenantID/users", element: wrap(<TenantUsers />) },
  { path: "/tenant/:tenantID/alerts", element: wrap(<Alerts />) },
  { path: "/tenant/:tenantID/downloads", element: wrap(<AgentDownloads />) },
  { path: "/tenant/:tenantID/reports", element: wrap(<Reports />) },
  { path: "/tenant/:tenantID/inventory", element: wrap(<Inventory />) },
  { path: "/tenant/:tenantID/support", element: wrap(<Support />) }
]);

createRoot(document.getElementById("root")).render(
  <React.StrictMode>
    <RouterProvider router={router} />
  </React.StrictMode>
);
