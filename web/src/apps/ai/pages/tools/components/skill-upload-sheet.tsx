import { useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { useQueryClient } from "@tanstack/react-query"
import { Upload, X, FileText } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
  Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription, SheetFooter,
} from "@/components/ui/sheet"
import { TOKEN_KEY } from "@/lib/constants"

interface SkillUploadSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function SkillUploadSheet({ open, onOpenChange }: SkillUploadSheetProps) {
  const { t } = useTranslation(["ai", "common"])
  const queryClient = useQueryClient()
  const inputRef = useRef<HTMLInputElement>(null)
  const [file, setFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [dragOver, setDragOver] = useState(false)

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const selected = e.target.files?.[0] ?? null
    setFile(selected)
    if (inputRef.current) inputRef.current.value = ""
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault()
    setDragOver(false)
    const dropped = e.dataTransfer.files[0]
    if (dropped && dropped.name.endsWith(".tar.gz")) {
      setFile(dropped)
    }
  }

  async function handleUpload() {
    if (!file) return
    setUploading(true)
    try {
      const formData = new FormData()
      formData.append("file", file)
      const token = localStorage.getItem(TOKEN_KEY)
      const res = await fetch("/api/v1/ai/skills/upload", {
        method: "POST",
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        body: formData,
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({})) as { message?: string }
        throw new Error(body.message ?? res.statusText)
      }
      queryClient.invalidateQueries({ queryKey: ["ai-skills"] })
      toast.success(t("ai:tools.skills.uploadSuccess"))
      setFile(null)
      onOpenChange(false)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err))
    } finally {
      setUploading(false)
    }
  }

  function handleOpenChange(val: boolean) {
    if (!uploading) {
      setFile(null)
      onOpenChange(val)
    }
  }

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetContent className="sm:max-w-lg overflow-y-auto">
        <SheetHeader>
          <SheetTitle>{t("ai:tools.skills.upload")}</SheetTitle>
          <SheetDescription className="sr-only">
            {t("ai:tools.skills.upload")}
          </SheetDescription>
        </SheetHeader>
        <div className="flex flex-1 flex-col gap-5 px-4">
          <div
            className={`relative flex flex-col items-center justify-center gap-3 rounded-lg border-2 border-dashed p-8 transition-colors cursor-pointer
              ${dragOver ? "border-primary bg-primary/5" : "border-muted-foreground/25 hover:border-muted-foreground/50"}`}
            onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
            onDragLeave={() => setDragOver(false)}
            onDrop={handleDrop}
            onClick={() => inputRef.current?.click()}
          >
            <Upload className="h-8 w-8 text-muted-foreground" />
            <div className="text-center">
              <p className="text-sm font-medium">{t("ai:tools.skills.dropHere")}</p>
              <p className="text-xs text-muted-foreground mt-1">.tar.gz</p>
            </div>
            <input
              ref={inputRef}
              type="file"
              accept=".tar.gz,.tgz,application/gzip"
              className="hidden"
              onChange={handleFileChange}
            />
          </div>

          {file && (
            <div className="flex items-center gap-2 rounded-md border bg-muted/30 px-3 py-2">
              <FileText className="h-4 w-4 text-muted-foreground shrink-0" />
              <span className="flex-1 truncate text-sm">{file.name}</span>
              <span className="text-xs text-muted-foreground shrink-0">{formatBytes(file.size)}</span>
              <button
                type="button"
                className="rounded p-0.5 hover:bg-accent text-muted-foreground"
                onClick={(e) => { e.stopPropagation(); setFile(null) }}
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </div>
          )}
        </div>
        <SheetFooter className="px-4">
          <Button
            size="sm"
            disabled={!file || uploading}
            onClick={handleUpload}
          >
            {uploading ? t("ai:tools.skills.uploading") : t("ai:tools.skills.uploadBtn")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
