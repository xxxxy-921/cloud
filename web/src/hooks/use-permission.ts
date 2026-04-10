import { useMenuStore } from "@/stores/menu"

export function usePermission(code: string): boolean {
  return useMenuStore((s) => s.permissions.includes(code))
}
