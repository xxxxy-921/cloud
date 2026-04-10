import { useState, useEffect } from "react"
import { useNavigate } from "react-router"
import { useQuery } from "@tanstack/react-query"
import { PanelLeft, LogOut, KeyRound, ShieldCheck, ChevronDown } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useUiStore } from "@/stores/ui"
import { useAuthStore } from "@/stores/auth"
import { api, type SiteInfo } from "@/lib/api"
import { ChangePasswordDialog } from "@/components/change-password-dialog"
import { TwoFactorSetupDialog } from "@/components/two-factor-setup-dialog"
import { NotificationBell } from "@/components/notification-bell"
import { cn } from "@/lib/utils"

export function TopNav() {
  const toggleSidebar = useUiStore((s) => s.toggleSidebar)
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)
  const navigate = useNavigate()

  const { data: siteInfo } = useQuery({
    queryKey: ["site-info"],
    queryFn: () => api.get<SiteInfo>("/api/v1/site-info"),
    staleTime: 60_000,
  })

  const [pwdDialogOpen, setPwdDialogOpen] = useState(false)
  const [tfaDialogOpen, setTfaDialogOpen] = useState(false)

  // Listen for password-expired events from api.ts 409 interceptor
  useEffect(() => {
    function handleExpired(e: Event) {
      const msg = (e as CustomEvent).detail?.message || "密码已过期，请修改密码"
      toast.warning(msg)
      setPwdDialogOpen(true)
    }
    window.addEventListener("password-expired", handleExpired)
    return () => window.removeEventListener("password-expired", handleExpired)
  }, [])

  async function handleLogout() {
    await logout()
    navigate("/login", { replace: true })
  }

  return (
    <>
      <header
        className={cn(
          "fixed inset-x-0 top-0 z-30 flex h-14 items-center gap-3 border-b border-border/40 px-4",
          "bg-background backdrop-blur-2xl",
        )}
      >
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 text-muted-foreground"
          onClick={toggleSidebar}
        >
          <PanelLeft className="h-4 w-4" />
        </Button>

        <div className="flex items-center gap-2">
          {siteInfo?.hasLogo && (
            <img
              src="/api/v1/site-info/logo"
              alt="Logo"
              className="h-7 w-7 rounded object-contain"
            />
          )}
          <span className="text-base font-semibold tracking-tight text-foreground">
            {siteInfo?.appName ?? "Metis"}
          </span>
        </div>

        <div className="ml-auto flex items-center gap-2">
          <NotificationBell />
          {user && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button className="flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground">
                  <span>{user.username}</span>
                  <ChevronDown className="h-3.5 w-3.5" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-40">
                <DropdownMenuItem onClick={() => setPwdDialogOpen(true)}>
                  <KeyRound className="mr-2 h-4 w-4" />
                  修改密码
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setTfaDialogOpen(true)}>
                  <ShieldCheck className="mr-2 h-4 w-4" />
                  两步验证
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={handleLogout} className="text-destructive focus:text-destructive">
                  <LogOut className="mr-2 h-4 w-4" />
                  退出登录
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      </header>

      <ChangePasswordDialog open={pwdDialogOpen} onOpenChange={setPwdDialogOpen} />
      <TwoFactorSetupDialog
        open={tfaDialogOpen}
        onOpenChange={setTfaDialogOpen}
        enabled={user?.twoFactorEnabled ?? false}
      />
    </>
  )
}
