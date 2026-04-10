import { useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import QRCode from "react-qr-code"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

interface TwoFactorSetupDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  enabled: boolean
}

type Step = "idle" | "qr" | "backup"

interface SetupData {
  secret: string
  qrUri: string
}

export function TwoFactorSetupDialog({
  open,
  onOpenChange,
  enabled,
}: TwoFactorSetupDialogProps) {
  const queryClient = useQueryClient()
  const [step, setStep] = useState<Step>("idle")
  const [setupData, setSetupData] = useState<SetupData | null>(null)
  const [code, setCode] = useState("")
  const [backupCodes, setBackupCodes] = useState<string[]>([])
  const [error, setError] = useState("")

  const setupMutation = useMutation({
    mutationFn: () => api.post<SetupData>("/api/v1/auth/2fa/setup", {}),
    onSuccess: (data) => {
      setSetupData(data)
      setStep("qr")
      setError("")
    },
    onError: (err: Error) => setError(err.message),
  })

  const confirmMutation = useMutation({
    mutationFn: () => api.post<{ backupCodes: string[] }>("/api/v1/auth/2fa/confirm", { code }),
    onSuccess: (data) => {
      setBackupCodes(data.backupCodes)
      setStep("backup")
      setError("")
      queryClient.invalidateQueries({ queryKey: ["auth", "me"] })
    },
    onError: (err: Error) => setError(err.message),
  })

  const disableMutation = useMutation({
    mutationFn: () => api.delete("/api/v1/auth/2fa"),
    onSuccess: () => {
      toast.success("两步验证已关闭")
      queryClient.invalidateQueries({ queryKey: ["auth", "me"] })
      handleClose()
    },
    onError: (err: Error) => toast.error(err.message),
  })

  function handleClose() {
    setStep("idle")
    setSetupData(null)
    setCode("")
    setBackupCodes([])
    setError("")
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>两步验证</DialogTitle>
          <DialogDescription>
            {enabled ? "管理两步验证设置" : "启用 TOTP 两步验证以增强账户安全"}
          </DialogDescription>
        </DialogHeader>

        {/* Idle: show enable/disable */}
        {step === "idle" && (
          <div className="space-y-4">
            {enabled ? (
              <div className="space-y-3">
                <p className="text-sm text-muted-foreground">
                  两步验证已启用。关闭后登录时将不再需要验证码。
                </p>
                <Button
                  variant="destructive"
                  onClick={() => disableMutation.mutate()}
                  disabled={disableMutation.isPending}
                >
                  {disableMutation.isPending ? "关闭中..." : "关闭两步验证"}
                </Button>
              </div>
            ) : (
              <div className="space-y-3">
                <p className="text-sm text-muted-foreground">
                  启用两步验证后，每次登录都需要输入验证器应用生成的验证码。
                </p>
                <Button
                  onClick={() => setupMutation.mutate()}
                  disabled={setupMutation.isPending}
                >
                  {setupMutation.isPending ? "生成中..." : "开始设置"}
                </Button>
              </div>
            )}
          </div>
        )}

        {/* QR code step */}
        {step === "qr" && setupData && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              使用验证器应用（如 Google Authenticator、Authy）扫描下方二维码：
            </p>
            <div className="flex justify-center rounded-lg border bg-white p-4">
              <QRCode value={setupData.qrUri} size={200} />
            </div>
            <details className="text-xs text-muted-foreground">
              <summary className="cursor-pointer">无法扫码？手动输入密钥</summary>
              <code className="mt-1 block break-all rounded bg-muted p-2 font-mono text-xs">
                {setupData.secret}
              </code>
            </details>
            <div className="space-y-2">
              <Label htmlFor="totp-code">输入验证码以确认</Label>
              <Input
                id="totp-code"
                placeholder="6 位验证码"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                autoComplete="one-time-code"
              />
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
            <Button
              className="w-full"
              onClick={() => confirmMutation.mutate()}
              disabled={!code || confirmMutation.isPending}
            >
              {confirmMutation.isPending ? "验证中..." : "确认启用"}
            </Button>
          </div>
        )}

        {/* Backup codes step */}
        {step === "backup" && (
          <div className="space-y-4">
            <p className="text-sm font-medium text-green-600">
              两步验证已启用！
            </p>
            <p className="text-sm text-muted-foreground">
              请妥善保存以下恢复码。当无法使用验证器时，可以使用恢复码登录。每个恢复码只能使用一次。
            </p>
            <div className="grid grid-cols-2 gap-2 rounded-lg border bg-muted p-3">
              {backupCodes.map((c) => (
                <code key={c} className="font-mono text-sm">
                  {c}
                </code>
              ))}
            </div>
            <Button className="w-full" onClick={handleClose}>
              我已保存，完成
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
