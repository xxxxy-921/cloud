import { FileQuestion } from "lucide-react"
import { Link } from "react-router"
import { Button } from "@/components/ui/button"

export default function NotFoundPage() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="flex flex-col items-center gap-4 text-muted-foreground">
        <FileQuestion className="h-16 w-16" />
        <h1 className="text-4xl font-bold text-foreground">404</h1>
        <p className="text-sm">页面不存在</p>
        <Button asChild variant="outline">
          <Link to="/">返回首页</Link>
        </Button>
      </div>
    </div>
  )
}
