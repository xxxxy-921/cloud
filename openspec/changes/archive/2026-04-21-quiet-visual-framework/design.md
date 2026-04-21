## Context

当前前端使用 `workspace-*` CSS 类体系定义工作区视觉风格。该体系采用毛玻璃设计（backdrop-blur 12-18px、白色渐变背景、双层 box-shadow、color-mix 边框），视觉重量较高。

供应商详情页（`providers/[id].tsx`）绕开了这套体系，直接使用 Tailwind 原生类（`bg-background/40`、`border-border/50`）实现了更安静的视觉效果，被确认为目标风格。

继承关系：`globals.css` workspace-* → `card.tsx` → `data-table.tsx` → 19+ 页面文件。改源头即可全局生效。

## Goals / Non-Goals

**Goals:**
- 将 workspace-* 类从毛玻璃风格改为安静轻量风格，对齐供应商详情页视觉语言
- 通过 CSS 框架继承实现一处改、处处生效
- 保持容器的结构作用（布局、间距、圆角），只去掉视觉装饰（blur、渐变、阴影）
- 更新 DESIGN.md 记录新的视觉基线

**Non-Goals:**
- 不改动任何组件文件（card.tsx、data-table.tsx、table.tsx）
- 不改动任何页面文件——全部通过 CSS 继承自动生效
- 不涉及 auth 系列类（auth-stage、auth-panel-glass 等登录页样式）
- 不涉及 install 系列类（安装向导样式）
- 不改动 body 背景渐变——它是安静风格的纵深基底

## Decisions

### D1: 在 CSS 源头改，不逐页迁移

**选择**: 重写 `globals.css` 中 6 个 workspace-* 类的值

**备选方案**:
- B: 新建轻量类（如 `workspace-surface-quiet`），逐页替换 → 工作量大 19+ 文件，且产生两套类共存的混乱
- C: 废弃 workspace-* 体系，全部改用 Tailwind inline → 失去框架控制力

**理由**: workspace-* 就是设计来做全局视觉控制的抽象层。改它而不是绕开它，才是框架思维。

### D2: 各类目标值

| 类名 | 去掉 | 保留/新增 |
|------|------|----------|
| `workspace-page-header` | blur, gradient, shadow, border, rounded | flex 布局 + padding + `border-b border-border/50 pb-5` |
| `workspace-surface` | blur, gradient, shadow | `background: oklch(... / 0.4)` + `border: 1px solid oklch(... / 0.5)` |
| `workspace-table-card` | blur, gradient, shadow | 同 workspace-surface，保留 rounded |
| `workspace-table-toolbar` | blur, gradient, shadow, border | `border-b border-border/32` 分隔，纯结构 |
| `workspace-toolbar-input` | shadow | `border-color: oklch(... / 0.5)` + `background: transparent` |
| `workspace-panel` | blur, gradient | `border-right: 1px solid oklch(... / 0.5)` + `background: oklch(... / 0.6)` |

### D3: 保留 body 背景渐变

**理由**: 容器变透明后，body 的微妙纵深渐变会隐约透出来，增加空间感。这是"安静"的基底，不是噪音。

### D4: workspace-shell-bg 保留但轻量化

当前有 radial-gradient 和 linear-gradient。保留基本背景色混合，去掉多余渐变层，让它更纯净。

## Risks / Trade-offs

- **一刀切风险**: 所有页面同时变化，某些场景（如 Settings 卡片、ITSM 工单详情）可能需要微调 → 通过全站视觉回归验证发现问题
- **侧边栏边界感**: workspace-panel 去掉 blur/shadow 后可能与内容区分不清 → 保留 border-right 和轻微背景色差
- **Card 内容对比度**: workspace-surface 变半透明后，Card 与页面背景的区分可能不够 → 保留弱边框 + 略高于背景的不透明度
