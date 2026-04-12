## Why

当前 AI 聊天界面采用传统的气泡对话设计（左右分栏、圆角气泡、头像图标），这种设计在 2023 年流行，但已显过时。现代 AI 产品（ChatGPT、Claude、Perplexity）已演进为更极简的文档流式设计，减少视觉噪音，让用户聚焦于内容本身。

## What Changes

- **移除气泡样式**：用户和 AI 消息不再使用圆角气泡包裹
- **改为单栏布局**：放弃左右分栏，所有内容左对齐，最大化阅读空间
- **优化排版层次**：标题、列表、代码块、表格采用现代文档排版风格
- **重新设计输入框**：底部固定、多行支持、自动增高，参考 Claude Composer
- **改造侧边栏**：改为可收起式或顶部下拉，释放主内容区空间
- **升级消息操作**：复制、重新生成、赞/踩改为悬浮或底部工具栏形式

## Capabilities

### New Capabilities
- `ai-chat-ui`: AI 聊天界面视觉重设计，包含消息展示、输入框、侧边栏的交互规范

### Modified Capabilities
- *(none - this is purely UI implementation change)*

## Impact

- **前端文件**: `web/src/apps/ai/pages/chat/` 目录下所有组件
- **依赖**: 无新增依赖，使用现有 Tailwind CSS 和 shadcn/ui 组件
- **API**: 无 API 变更，纯前端视觉改造
- **用户体验**: 新旧界面切换需要用户适应，但操作逻辑保持不变
