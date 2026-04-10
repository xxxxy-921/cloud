import { useState } from "react"
import { useNavigate, useLocation, Navigate } from "react-router"
import { useAuthStore } from "@/stores/auth"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

export function Component() {
  const navigate = useNavigate()
  const location = useLocation()
  const oauthLogin = useAuthStore((s) => s.oauthLogin)
  const user = useAuthStore((s) => s.user)

  const twoFactorToken = (location.state as any)?.twoFactorToken as string | undefined

  const [code, setCode] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)
  const [useBackup, setUseBackup] = useState(false)

  // Already logged in or no token
  if (user) return <Navigate to="/" replace />
  if (!twoFactorToken) return <Navigate to="/login" replace />

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError("")
    setLoading(true)

    try {
      const res = await fetch("/api/v1/auth/2fa/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ twoFactorToken, code }),
      })

      const body = await res.json()
      if (!res.ok || body.code !== 0) {
        throw new Error(body.message || "验证失败")
      }

      await oauthLogin(body.data)
      navigate("/", { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : "验证失败")
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="w-full max-w-sm space-y-6 px-4">
        <div className="space-y-2 text-center">
          <h1 className="text-2xl font-semibold tracking-tight">两步验证</h1>
          <p className="text-sm text-muted-foreground">
            {useBackup
              ? "请输入恢复码"
              : "请输入验证器应用中的 6 位验证码"}
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="code">{useBackup ? "恢复码" : "验证码"}</Label>
            <Input
              id="code"
              placeholder={useBackup ? "请输入 8 位恢复码" : "请输入 6 位验证码"}
              value={code}
              onChange={(e) => setCode(e.target.value)}
              required
              autoFocus
              autoComplete="one-time-code"
            />
          </div>

          {error && <p className="text-sm text-destructive">{error}</p>}

          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? "验证中..." : "验证"}
          </Button>
        </form>

        <div className="text-center">
          <button
            type="button"
            className="text-sm text-muted-foreground hover:text-primary hover:underline"
            onClick={() => {
              setUseBackup(!useBackup)
              setCode("")
              setError("")
            }}
          >
            {useBackup ? "使用验证码" : "使用恢复码"}
          </button>
        </div>
      </div>
    </div>
  )
}
