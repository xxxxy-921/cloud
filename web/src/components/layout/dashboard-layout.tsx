import { Outlet, useLocation } from "react-router"
import { TopNav } from "./top-nav"
import { Sidebar } from "./sidebar"
import { useUiStore } from "@/stores/ui"
import { cn } from "@/lib/utils"

export function DashboardLayout() {
  const collapsed = useUiStore((s) => s.sidebarCollapsed)
  const location = useLocation()

  // Check if current route is chat session page - needs fullscreen layout
  // /ai/chat (agent selection) needs padding, /ai/chat/:sid (chat session) does not
  const isChatSessionRoute = /^\/ai\/chat\/\d+/.test(location.pathname)

  return (
    <div className="workspace-shell-bg min-h-screen">
      <TopNav />
      <Sidebar />
      <main
        className={cn(
          "pt-14 transition-all duration-200",
          collapsed ? "pl-12" : "pl-52",
          isChatSessionRoute && "h-screen overflow-hidden",
        )}
      >
        <div className={cn(
          "flex h-full flex-col",
          !isChatSessionRoute && "p-4 sm:p-5 lg:p-6",
        )}>
          <Outlet />
        </div>
      </main>
    </div>
  )
}
