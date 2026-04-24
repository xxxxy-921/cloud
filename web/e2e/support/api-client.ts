import type { Page } from "@playwright/test"

const accessTokenKey = "metis_access_token"

export type APIResult<T> = {
  ok: boolean
  status: number
  body: {
    code: number
    message: string
    data: T
  }
}

export async function apiRequest<T>(page: Page, method: string, path: string, body?: unknown) {
  return await page.evaluate(
    async ({ accessTokenKey, body, method, path }) => {
      const token = window.localStorage.getItem(accessTokenKey)
      const headers: Record<string, string> = {
        "Content-Type": "application/json",
      }
      if (token) {
        headers.Authorization = `Bearer ${token}`
      }
      const response = await window.fetch(path, {
        method,
        headers,
        body: body === undefined ? undefined : JSON.stringify(body),
      })
      return {
        ok: response.ok,
        status: response.status,
        body: await response.json(),
      }
    },
    { accessTokenKey, body, method, path },
  ) as APIResult<T>
}
