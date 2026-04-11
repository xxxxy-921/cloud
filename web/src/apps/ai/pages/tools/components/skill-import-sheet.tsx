import { useState } from "react"
import { useTranslation } from "react-i18next"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription, SheetFooter,
} from "@/components/ui/sheet"
import { Label } from "@/components/ui/label"

interface SkillImportSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function SkillImportSheet({ open, onOpenChange }: SkillImportSheetProps) {
  const { t } = useTranslation(["ai", "common"])
  const queryClient = useQueryClient()
  const [url, setUrl] = useState("")

  const importMutation = useMutation({
    mutationFn: (githubUrl: string) =>
      api.post("/api/v1/ai/skills/import-github", { url: githubUrl }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ai-skills"] })
      onOpenChange(false)
      setUrl("")
      toast.success(t("ai:tools.skills.importSuccess"))
    },
    onError: (err) => toast.error(err.message),
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!url.trim()) return
    importMutation.mutate(url.trim())
  }

  function handleOpenChange(val: boolean) {
    if (!importMutation.isPending) {
      setUrl("")
      onOpenChange(val)
    }
  }

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetContent className="sm:max-w-lg overflow-y-auto">
        <SheetHeader>
          <SheetTitle>{t("ai:tools.skills.importGitHub")}</SheetTitle>
          <SheetDescription className="sr-only">
            {t("ai:tools.skills.importGitHub")}
          </SheetDescription>
        </SheetHeader>
        <form onSubmit={handleSubmit} className="flex flex-1 flex-col gap-5 px-4">
          <div className="space-y-2">
            <Label>{t("ai:tools.skills.githubUrl")}</Label>
            <Input
              placeholder={t("ai:tools.skills.githubUrlPlaceholder")}
              value={url}
              onChange={(e) => setUrl(e.target.value)}
            />
          </div>
          <SheetFooter>
            <Button
              type="submit"
              size="sm"
              disabled={!url.trim() || importMutation.isPending}
            >
              {importMutation.isPending ? t("common:saving") : t("ai:tools.skills.importGitHub")}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}
