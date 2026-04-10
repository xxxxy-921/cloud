import { Outlet } from "react-router"
import { TopNav } from "./top-nav"
import { Sidebar } from "./sidebar"
import { Header } from "./header"
import { useUiStore } from "@/stores/ui"
import { cn } from "@/lib/utils"

export function DashboardLayout() {
  const collapsed = useUiStore((s) => s.sidebarCollapsed)

  return (
    <div className="min-h-screen bg-background">
      <TopNav />
      <Sidebar />
      <main
        className={cn(
          "pt-14 transition-all duration-200",
          collapsed ? "pl-12" : "pl-52",
        )}
      >
        <Header />
        <div className="flex flex-col p-6">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
