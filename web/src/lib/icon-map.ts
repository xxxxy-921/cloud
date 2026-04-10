import {
  Home,
  Settings,
  Users,
  Shield,
  Menu,
  Sliders,
  Wrench,
  FileText,
  Folder,
  LayoutDashboard,
  FolderOpen,
  MousePointerClick,
  KeyRound,
  UserCog,
  Database,
  Bell,
  BarChart3,
  Globe,
  Lock,
  type LucideIcon,
} from "lucide-react"

const iconMap: Record<string, LucideIcon> = {
  Home,
  Settings,
  Users,
  Shield,
  Menu,
  Sliders,
  Wrench,
  FileText,
  Folder,
  FolderOpen,
  LayoutDashboard,
  MousePointerClick,
  KeyRound,
  UserCog,
  Database,
  Bell,
  BarChart3,
  Globe,
  Lock,
}

/** Fallback icon per menu type */
const typeFallback: Record<string, LucideIcon> = {
  directory: Folder,
  menu: FileText,
  button: MousePointerClick,
}

/**
 * Resolve a menu item's icon by name.
 * Falls back to a type-specific icon, then to FileText.
 */
export function getIcon(name: string | undefined, type?: string): LucideIcon {
  if (name && iconMap[name]) return iconMap[name]
  if (type && typeFallback[type]) return typeFallback[type]
  return FileText
}
