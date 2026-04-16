import { useEffect, useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet"
import { useTranslation } from "react-i18next"

interface RenewableLicense {
  id: number
  productName: string
  licenseeName: string
  registrationCode: string
  validUntil: string | null
}

interface RenewLicenseSheetProps {
  license: RenewableLicense | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function RenewLicenseSheet({ license, open, onOpenChange }: RenewLicenseSheetProps) {
  const { t } = useTranslation(["license", "common"])
  const queryClient = useQueryClient()
  const [validUntil, setValidUntil] = useState("")

  useEffect(() => {
    if (open && license) {
      queueMicrotask(() => {
        setValidUntil(license.validUntil ? license.validUntil.split("T")[0] : "")
      })
    }
  }, [open, license])

  const renewMutation = useMutation({
    mutationFn: () =>
      api.post(`/api/v1/license/licenses/${license!.id}/renew`, {
        validUntil: validUntil || null,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["license-licenses"] })
      queryClient.invalidateQueries({ queryKey: ["license-license", String(license!.id)] })
      onOpenChange(false)
      toast.success(t("license:licenses.renewSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!license) return
    renewMutation.mutate()
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>{t("license:licenses.renewTitle")}</SheetTitle>
          <SheetDescription className="sr-only">{t("license:licenses.renewDesc")}</SheetDescription>
        </SheetHeader>
        <form onSubmit={handleSubmit} className="flex flex-1 flex-col gap-4 px-4">
          {/* Product (read-only) */}
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">{t("license:licenses.product")}</Label>
            <Input value={license ? license.productName : "-"} disabled />
          </div>

          {/* Licensee (read-only) */}
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">{t("license:licenses.licensee")}</Label>
            <Input value={license ? license.licenseeName : "-"} disabled />
          </div>

          {/* Registration Code (read-only) */}
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">{t("license:licenses.registrationCode")}</Label>
            <Input value={license?.registrationCode ?? ""} disabled />
          </div>

          {/* Valid Until */}
          <div className="space-y-1.5">
            <Label className="text-xs text-muted-foreground">{t("license:licenses.validUntilDate")}</Label>
            <Input
              type="date"
              value={validUntil}
              onChange={(e) => setValidUntil(e.target.value)}
              placeholder={t("license:licenses.emptyForPermanent")}
            />
          </div>

          <SheetFooter>
            <Button
              type="submit"
              size="sm"
              className="h-8 rounded-lg px-3"
              disabled={renewMutation.isPending}
            >
              {renewMutation.isPending ? t("common:processing") : t("common:confirm")}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}
