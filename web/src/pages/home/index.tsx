import { Home } from "lucide-react"

export default function HomePage() {
  return (
    <div className="flex flex-1 items-center justify-center">
      <div className="flex flex-col items-center gap-4 text-muted-foreground">
        <Home className="h-12 w-12" />
        <h1 className="text-2xl font-semibold text-foreground">欢迎使用 Metis</h1>
        <p className="text-sm">选择左侧菜单开始操作</p>
      </div>
    </div>
  )
}
