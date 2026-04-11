import { useTranslation } from "react-i18next"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { BuiltinToolsTab } from "./components/builtin-tools-tab"
import { MCPServersTab } from "./components/mcp-servers-tab"
import { SkillsTab } from "./components/skills-tab"

export function Component() {
  const { t } = useTranslation("ai")

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">{t("tools.title")}</h2>
      <Tabs defaultValue="builtin">
        <TabsList>
          <TabsTrigger value="builtin">{t("tools.tabs.builtinTools")}</TabsTrigger>
          <TabsTrigger value="mcp">{t("tools.tabs.mcpServers")}</TabsTrigger>
          <TabsTrigger value="skills">{t("tools.tabs.skills")}</TabsTrigger>
        </TabsList>
        <TabsContent value="builtin" className="mt-4">
          <BuiltinToolsTab />
        </TabsContent>
        <TabsContent value="mcp" className="mt-4">
          <MCPServersTab />
        </TabsContent>
        <TabsContent value="skills" className="mt-4">
          <SkillsTab />
        </TabsContent>
      </Tabs>
    </div>
  )
}
