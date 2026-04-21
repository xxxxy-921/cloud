## Why

当前 workspace-* CSS 类体系使用毛玻璃风格（backdrop-blur、渐变背景、双层阴影），视觉重量过高。已落地的供应商详情页证明了轻量化方案（半透明背景、弱边框、无 blur/shadow）更安静、更聚焦内容。两套视觉语言在共存，需要统一到"安静"方向。

## What Changes

- 重写 `workspace-page-header` 类：去掉 blur/gradient/shadow/rounded，改为纯结构布局 + `border-b` 分隔
- 重写 `workspace-surface` 类（Card 基底）：从毛玻璃改为 `bg-background/40` + `border-border/50`，无 blur/shadow
- 重写 `workspace-table-card` 类（DataTableCard 基底）：同上轻量化
- 重写 `workspace-table-toolbar` 类（DataTableToolbar 基底）：去掉所有装饰，纯结构容器
- 重写 `workspace-toolbar-input` 类：极简化，去掉 shadow
- 轻量化 `workspace-panel` 类（侧边栏）：去掉 blur/gradient，保留轻微边界感
- 同步更新 `DESIGN.md`，将安静视觉语言记录为新基线

## Capabilities

### New Capabilities

_无新增能力_

### Modified Capabilities

- `shared-ui-patterns`: 全局视觉重量从毛玻璃降级为轻量透明风格，workspace-* 类的视觉定义发生变化
- `ai-provider-detail-page`: 供应商详情页成为全局视觉基线参考，无代码改动但设计基线地位需记录

## Impact

- **CSS**: 仅修改 `web/src/styles/globals.css` 中 6 个 workspace-* 类
- **组件**: `card.tsx`、`data-table.tsx`、`table.tsx` 无需改动，通过 CSS 继承自动生效
- **页面**: 全站 19+ 使用这些类的文件自动获得新视觉，无需逐页修改
- **风险**: 一刀切改动，需要全站视觉回归验证；个别页面可能需要微调
