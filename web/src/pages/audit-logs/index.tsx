import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { AuthTab } from "./auth-tab"
import { OperationTab } from "./operation-tab"

export function Component() {
  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">审计日志</h2>

      <Tabs defaultValue="auth">
        <TabsList>
          <TabsTrigger value="auth">登录活动</TabsTrigger>
          <TabsTrigger value="operation">操作记录</TabsTrigger>
        </TabsList>
        <TabsContent value="auth">
          <AuthTab />
        </TabsContent>
        <TabsContent value="operation">
          <OperationTab />
        </TabsContent>
      </Tabs>
    </div>
  )
}
