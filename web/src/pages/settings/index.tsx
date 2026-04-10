import { useQuery } from "@tanstack/react-query"
import { api, type SiteInfo } from "@/lib/api"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { SiteNameCard } from "./site-name-card"
import { LogoCard } from "./logo-card"
import { SecurityCard } from "./security-card"
import { SchedulerCard } from "./scheduler-card"
import { ConnectionsCard } from "./connections-card"

export function Component() {
  const { data, isLoading } = useQuery({
    queryKey: ["site-info"],
    queryFn: () => api.get<SiteInfo>("/api/v1/site-info"),
  })

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center text-muted-foreground">
        加载中...
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold">系统设置</h2>
      <Tabs defaultValue="site">
        <TabsList>
          <TabsTrigger value="site">站点信息</TabsTrigger>
          <TabsTrigger value="security">安全设置</TabsTrigger>
          <TabsTrigger value="scheduler">任务设置</TabsTrigger>
          <TabsTrigger value="connections">账号关联</TabsTrigger>
        </TabsList>
        <TabsContent value="site" className="space-y-6 mt-4">
          <SiteNameCard appName={data?.appName ?? "Metis"} />
          <LogoCard hasLogo={data?.hasLogo ?? false} />
        </TabsContent>
        <TabsContent value="security" className="mt-4">
          <SecurityCard />
        </TabsContent>
        <TabsContent value="scheduler" className="mt-4">
          <SchedulerCard />
        </TabsContent>
        <TabsContent value="connections" className="mt-4">
          <ConnectionsCard />
        </TabsContent>
      </Tabs>
    </div>
  )
}
