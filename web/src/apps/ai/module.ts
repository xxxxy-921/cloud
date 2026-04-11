import { registerApp } from "@/apps/registry"
import { registerTranslations } from "@/i18n"
import zhCN from "./locales/zh-CN.json"
import en from "./locales/en.json"

registerTranslations("ai", { "zh-CN": zhCN, en })

registerApp({
  name: "ai",
  routes: [
    {
      path: "ai/providers",
      children: [
        {
          index: true,
          lazy: () => import("./pages/providers/index"),
        },
      ],
    },
  ],
})
