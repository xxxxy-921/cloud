import { useEffect, useState } from "react"
import { useNavigate, useSearchParams } from "react-router"
import { useAuthStore } from "@/stores/auth"
import { api } from "@/lib/api"

export function Component() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const oauthLogin = useAuthStore((s) => s.oauthLogin)
  const [error, setError] = useState("")

  useEffect(() => {
    const code = searchParams.get("code")
    const state = searchParams.get("state")

    if (!code || !state) {
      navigate("/login", { replace: true })
      return
    }

    // Extract provider from state - we pass it via the callback URL or detect it
    // The backend will validate the state and determine the provider
    handleCallback(code, state)
  }, [])

  async function handleCallback(code: string, state: string) {
    try {
      // Try each known provider (the state contains the provider info server-side)
      // We need the provider in the request body, so we store it in sessionStorage
      const provider = sessionStorage.getItem("oauth_provider") || "github"
      sessionStorage.removeItem("oauth_provider")

      const data = await api.post<{
        accessToken: string
        refreshToken: string
        expiresIn: number
        permissions: string[]
      }>("/auth/oauth/callback", { provider, code, state })

      await oauthLogin(data)
      navigate("/", { replace: true })
    } catch (err) {
      const msg = err instanceof Error ? err.message : "OAuth login failed"
      setError(msg)
      setTimeout(() => navigate("/login", { replace: true, state: { error: msg } }), 3000)
    }
  }

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="text-center space-y-2">
          <p className="text-sm text-destructive">{error}</p>
          <p className="text-xs text-muted-foreground">正在跳转到登录页...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <p className="text-sm text-muted-foreground">正在登录...</p>
    </div>
  )
}
