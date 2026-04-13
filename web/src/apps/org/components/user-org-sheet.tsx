import { useTranslation } from "react-i18next"
import { useQuery } from "@tanstack/react-query"
import { api } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet"

interface UserOrgSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  userId: number | null
  username: string
  email: string
}

interface UserPositionItem {
  id: number
  userId: number
  departmentId: number
  positionId: number
  isPrimary: boolean
  department?: { id: number; name: string }
  position?: { id: number; name: string }
}

export function UserOrgSheet({ open, onOpenChange, userId, username, email }: UserOrgSheetProps) {
  const { t } = useTranslation(["org", "common"])

  const { data: positions, isLoading } = useQuery({
    queryKey: ["user-org-positions", userId],
    queryFn: async () => {
      const res = await api.get<{ items: UserPositionItem[] }>(`/api/v1/org/users/${userId}/positions`)
      return res.items
    },
    enabled: open && !!userId,
  })

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="gap-0 p-0 sm:max-w-md">
        <SheetHeader className="border-b px-6 py-5">
          <SheetTitle>{t("org:assignments.orgInfo")}</SheetTitle>
          <SheetDescription className="sr-only">
            {t("org:assignments.orgInfo")}
          </SheetDescription>
        </SheetHeader>
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
          <div className="flex-1 space-y-5 overflow-auto px-6 py-6">
            <div className="flex items-center gap-3">
              <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-muted text-sm font-semibold text-foreground/80">
                {username.charAt(0).toUpperCase()}
              </div>
              <div className="min-w-0">
                <p className="text-sm font-medium text-foreground">{username}</p>
                {email && <p className="truncate text-xs text-muted-foreground">{email}</p>}
              </div>
            </div>

            <div className="space-y-3">
              <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                {t("org:assignments.orgAssignments")}
              </p>
              {isLoading ? (
                <p className="text-sm text-muted-foreground">{t("common:loading")}</p>
              ) : !positions || positions.length === 0 ? (
                <p className="text-sm text-muted-foreground">{t("org:assignments.noAssignments")}</p>
              ) : (
                <div className="space-y-2">
                  {positions.map((item) => (
                    <div
                      key={item.id}
                      className="flex items-center justify-between rounded-lg border bg-card px-3 py-2.5"
                    >
                      <div className="space-y-0.5">
                        <p className="text-sm font-medium text-foreground">{item.department?.name ?? "-"}</p>
                        <p className="text-xs text-muted-foreground">{item.position?.name ?? "-"}</p>
                      </div>
                      {item.isPrimary ? (
                        <Badge variant="default" className="shrink-0">{t("org:assignments.primary")}</Badge>
                      ) : (
                        <Badge variant="outline" className="shrink-0">{t("org:assignments.secondary")}</Badge>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  )
}
