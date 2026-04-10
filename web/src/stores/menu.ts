import { create } from "zustand"
import { api } from "@/lib/api"

export interface MenuItem {
  id: number
  parentId: number | null
  name: string
  type: "directory" | "menu" | "button"
  path: string
  icon: string
  permission: string
  sort: number
  isHidden: boolean
  children: MenuItem[]
}

interface MenuState {
  menuTree: MenuItem[]
  permissions: string[]
  initialized: boolean

  init: () => Promise<void>
  clear: () => void
  hasPermission: (code: string) => boolean
}

export const useMenuStore = create<MenuState>((set, get) => ({
  menuTree: [],
  permissions: [],
  initialized: false,

  init: async () => {
    try {
      const data = await api.get<{ menus: MenuItem[]; permissions: string[] }>(
        "/api/v1/menus/user-tree",
      )
      set({
        menuTree: data.menus || [],
        permissions: data.permissions || [],
        initialized: true,
      })
    } catch {
      set({ menuTree: [], permissions: [], initialized: true })
    }
  },

  clear: () => {
    set({ menuTree: [], permissions: [], initialized: false })
  },

  hasPermission: (code: string) => {
    return get().permissions.includes(code)
  },
}))
