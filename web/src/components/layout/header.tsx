import { useMemo } from "react"
import { useLocation } from "react-router"
import { useMenuStore, type MenuItem } from "@/stores/menu"
import { Separator } from "@/components/ui/separator"

function buildPathLabels(menuTree: MenuItem[]): Record<string, string> {
  const labels: Record<string, string> = {}
  function walk(items: MenuItem[]) {
    for (const m of items) {
      if (m.path) {
        // Use full path as key so cumulative breadcrumb lookup works
        labels[m.path] = m.name
      }
      if (m.children) walk(m.children)
    }
  }
  walk(menuTree)
  return labels
}

export function Header() {
  const { pathname } = useLocation()
  const menuTree = useMenuStore((s) => s.menuTree)

  const pathLabels = useMemo(() => buildPathLabels(menuTree), [menuTree])

  const segments = pathname.split("/").filter(Boolean)
  const crumbs = [
    { label: "首页", path: "/" },
    ...segments.map((seg, i) => {
      const fullPath = "/" + segments.slice(0, i + 1).join("/")
      return {
        label: pathLabels[fullPath] ?? seg,
        path: fullPath,
      }
    }),
  ]

  if (crumbs.length <= 1) return null

  return (
    <div className="flex h-10 items-center gap-2 px-6 text-sm text-muted-foreground">
      {crumbs.map((crumb, i) => (
        <span key={crumb.path} className="flex items-center gap-2">
          {i > 0 && <Separator orientation="vertical" className="h-4" />}
          <span className={i === crumbs.length - 1 ? "text-foreground font-medium" : ""}>
            {crumb.label}
          </span>
        </span>
      ))}
    </div>
  )
}
