import { useEffect, useState } from "react"
import { createBrowserRouter, Navigate, RouterProvider } from "react-router"
import { DashboardLayout } from "@/components/layout/dashboard-layout"
import { PermissionGuard } from "@/components/permission-guard"
import { useAuthStore } from "@/stores/auth"
import { TwoFactorSetupDialog } from "@/components/two-factor-setup-dialog"
import { getAppRoutes } from "@/apps/registry"
// Pluggable app module imports — must be after registry is defined
import "@/apps/license/module"
import HomePage from "@/pages/home"
import LoginPage from "@/pages/login"
import NotFoundPage from "@/pages/not-found"

function AuthGuard() {
  const { user, initialized, requireTwoFactorSetup } = useAuthStore()
  const [tfaOpen, setTfaOpen] = useState(true)

  if (!initialized) {
    return (
      <div className="flex min-h-screen items-center justify-center text-muted-foreground">
        加载中...
      </div>
    )
  }

  if (!user) {
    return <Navigate to="/login" replace />
  }

  // Force 2FA setup when required by admin policy
  if (requireTwoFactorSetup && !user.twoFactorEnabled) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="text-center space-y-4">
          <h2 className="text-lg font-semibold">需要设置两步验证</h2>
          <p className="text-sm text-muted-foreground">
            系统管理员要求所有用户启用两步验证后才能继续使用。
          </p>
          <TwoFactorSetupDialog
            open={tfaOpen}
            onOpenChange={(open) => {
              setTfaOpen(open)
              // Re-open if user tries to close without enabling 2FA
              if (!open && !useAuthStore.getState().user?.twoFactorEnabled) {
                setTimeout(() => setTfaOpen(true), 100)
              }
              // Clear the flag once 2FA is enabled
              if (!open && useAuthStore.getState().user?.twoFactorEnabled) {
                useAuthStore.setState({ requireTwoFactorSetup: false })
              }
            }}
            enabled={false}
          />
        </div>
      </div>
    )
  }

  return <DashboardLayout />
}

const router = createBrowserRouter([
  {
    path: "/login",
    element: <LoginPage />,
  },
  {
    path: "/register",
    lazy: () => import("@/pages/register"),
  },
  {
    path: "/2fa",
    lazy: () => import("@/pages/two-factor"),
  },
  {
    path: "/oauth/callback",
    lazy: () => import("@/pages/oauth/callback"),
  },
  {
    path: "/sso/callback",
    lazy: () => import("@/pages/sso/callback"),
  },
  {
    element: <AuthGuard />,
    children: [
      { index: true, element: <HomePage /> },
      {
        path: "settings",
        lazy: () => import("@/pages/settings"),
      },
      {
        path: "users",
        element: <PermissionGuard permission="system:user:list" />,
        children: [
          {
            index: true,
            lazy: () => import("@/pages/users"),
          },
        ],
      },
      {
        path: "roles",
        element: <PermissionGuard permission="system:role:list" />,
        children: [
          {
            index: true,
            lazy: () => import("@/pages/roles"),
          },
        ],
      },
      {
        path: "menus",
        element: <PermissionGuard permission="system:menu:list" />,
        children: [
          {
            index: true,
            lazy: () => import("@/pages/menus"),
          },
        ],
      },
      {
        path: "sessions",
        element: <PermissionGuard permission="system:session:list" />,
        children: [
          {
            index: true,
            lazy: () => import("@/pages/sessions"),
          },
        ],
      },
      {
        path: "tasks",
        element: <PermissionGuard permission="system:task:list" />,
        children: [
          {
            index: true,
            lazy: () => import("@/pages/tasks"),
          },
          {
            path: ":name",
            lazy: () => import("@/pages/tasks/detail"),
          },
        ],
      },
      {
        path: "announcements",
        element: <PermissionGuard permission="system:announcement:list" />,
        children: [
          {
            index: true,
            lazy: () => import("@/pages/announcements"),
          },
        ],
      },
      {
        path: "channels",
        element: <PermissionGuard permission="system:channel:list" />,
        children: [
          {
            index: true,
            lazy: () => import("@/pages/channels"),
          },
        ],
      },
      {
        path: "auth-providers",
        element: <PermissionGuard permission="system:auth-provider:list" />,
        children: [
          {
            index: true,
            lazy: () => import("@/pages/auth-providers"),
          },
        ],
      },
      {
        path: "audit-logs",
        element: <PermissionGuard permission="system:audit-log:list" />,
        children: [
          {
            index: true,
            lazy: () => import("@/pages/audit-logs"),
          },
        ],
      },
      {
        path: "identity-sources",
        element: <PermissionGuard permission="system:identity-source:list" />,
        children: [
          {
            index: true,
            lazy: () => import("@/pages/identity-sources"),
          },
        ],
      },
      // Pluggable app routes
      ...getAppRoutes(),
    ],
  },
  { path: "*", element: <NotFoundPage /> },
])

function AppInit({ children }: { children: React.ReactNode }) {
  const init = useAuthStore((s) => s.init)
  const initialized = useAuthStore((s) => s.initialized)

  useEffect(() => {
    if (!initialized) {
      init()
    }
  }, [init, initialized])

  return <>{children}</>
}

export default function App() {
  return (
    <AppInit>
      <RouterProvider router={router} />
    </AppInit>
  )
}
