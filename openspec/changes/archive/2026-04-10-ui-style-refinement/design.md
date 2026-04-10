## Context

登录页已完成 2026 设计风格升级（紧凑、克制、降饱和），但系统内页仍使用 shadcn/ui 默认视觉：饱和靛蓝 primary、visible shadow、粗 focus ring。两者之间存在明显的风格割裂。

当前技术栈：Tailwind CSS 4 + shadcn/ui，所有 design tokens 通过 `globals.css` 的 `@theme` 定义，组件级样式在 `components/ui/*.tsx` 中。Token 驱动架构意味着修改少量变量即可全局生效。

## Goals / Non-Goals

**Goals:**
- 通过调整 design tokens 和基础组件，让内页视觉与登录页统一
- 主色降饱和度，达到成熟、沉稳的蓝色调
- 减少阴影和边框的视觉重量
- 统一 Input 组件的尺寸和圆角
- Badge 默认变体使用更柔和的色调
- DataTablePagination 样式现代化

**Non-Goals:**
- 不改变 Sidebar 结构、交互、宽度
- 不改变 TopNav 布局
- 不改变 Table 行间距（py-3 保持）
- 不引入暗色模式
- 不改变圆角策略（保持 rounded-xl 基调）
- 不调整字体

## Decisions

### Decision 1: 主色调整策略 — 降饱和度而非换色相

将 `--color-primary` 从 `oklch(0.488 0.185 264)` 调整为 `oklch(0.50 0.13 264)`。

**理由**: 保持蓝色色相不变（264），仅将 chroma 从 0.185 降至 ~0.13，lightness 微调至 0.50。这样所有使用 primary 的元素（按钮、Badge、focus ring、链接）自动变得沉稳，无需逐个组件修改。

**备选方案**: 换用深灰色（如登录页 CTA），但用户明确选择保留蓝调（方案 B）。

### Decision 2: Card 阴影策略 — border-only

Card 组件从 `shadow-sm` 改为无阴影，仅靠 border 界定边界。

**理由**: 登录页的玻璃面板仅用 border + 极淡投影，内页卡片不需要比它更重。`shadow-sm` 在浅色背景上制造了不必要的「浮起」感。移除后界面更扁平、更安静。

**备选方案**: 保留极淡阴影 `shadow-[0_1px_2px_rgba(0,0,0,0.04)]`，可在实施时视实际效果决定。

### Decision 3: Input 组件统一 — 对齐 auth-input 质感

内页 Input 高度从 `h-9`(36px) 调整为 `h-[2.375rem]`(38px)，圆角从 `rounded-md` 改为 `rounded-lg`，focus ring 从 3px 缩至 2px。

**理由**: 登录页 auth-input 使用 `h-[2.625rem]` + `rounded-xl`，内页不需要完全相同（内页信息密度更高），但需要在同一个视觉语言内。h-[2.375rem] + rounded-lg 是合理的折中。

### Decision 4: Badge 默认变体 — 使用淡底深字

Badge default variant 从 `bg-primary text-primary-foreground` 改为 `bg-primary/10 text-primary`（淡底 + 深字）。

**理由**: 全色底 Badge 在表格中过于抢眼。淡底方案视觉重量更轻，与 Notion/Linear 的标签风格一致。

### Decision 5: DataTablePagination — 去虚线

移除 `border-dashed`，改为 `border-t border-border/40`（顶部细线分隔）或无边框。

**理由**: 虚线边框是偏旧的设计语言，与整体克制风格不协调。

## Risks / Trade-offs

- **[primary 色值调整影响面广]** → 所有使用 bg-primary 的地方都会变化。通过 token 机制这是优势而非风险，但需要全局检查视觉一致性。
- **[Card 去阴影后层次感减弱]** → 如果内容区背景色与 Card 相近，可能缺乏区分。缓解：保留 border 提供边界感。
- **[Input 高度变化可能影响表单布局]** → 从 36px 到 38px 差异微小，但 inline 布局（如搜索栏）需要验证。
