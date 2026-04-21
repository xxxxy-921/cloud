## 1. CSS 框架层轻量化

- [x] 1.1 重写 `workspace-page-header`：去掉 border/background/box-shadow/backdrop-filter/rounded，改为 `border-b border-border/50 pb-5` 分隔
- [x] 1.2 重写 `workspace-surface`：去掉 gradient/box-shadow/backdrop-filter，改为半透明背景 + 弱边框
- [x] 1.3 重写 `workspace-table-card`：去掉 gradient/box-shadow/backdrop-filter，改为半透明背景 + 弱边框
- [x] 1.4 重写 `workspace-table-toolbar`：去掉 border/background/box-shadow/backdrop-filter，改为 `border-b border-border/32` 分隔
- [x] 1.5 重写 `workspace-toolbar-input`：去掉 box-shadow，改为透明背景 + 弱边框
- [x] 1.6 重写 `workspace-panel`：去掉 gradient/backdrop-filter，改为轻背景 + 右边框，可选保留极轻 shadow
- [x] 1.7 轻量化 `workspace-shell-bg`：简化渐变层，保留基本背景色

## 2. 验证与文档

- [x] 2.1 运行 `cd web && bun run build` 确认编译通过
- [x] 2.2 运行 `cd web && bun run lint` 确认无 lint 问题
- [x] 2.3 更新 DESIGN.md：将安静视觉语言记录为新基线，标记毛玻璃风格为废弃模式
