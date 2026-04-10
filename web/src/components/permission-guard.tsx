import { Outlet } from "react-router"
import { useMenuStore } from "@/stores/menu"

interface PermissionGuardProps {
  permission: string
  children?: React.ReactNode
}

export function PermissionGuard({ permission, children }: PermissionGuardProps) {
  const hasPermission = useMenuStore((s) => s.permissions.includes(permission))

  if (!hasPermission) {
    return (
      <div className="flex h-64 items-center justify-center text-muted-foreground">
        无权访问此页面
      </div>
    )
  }

  return children ?? <Outlet />
}
