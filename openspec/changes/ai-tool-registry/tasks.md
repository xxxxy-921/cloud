## 1. 数据模型

- [x] 1.1 新增 Tool model（ai_tools 表）：name, display_name, description, parameters_schema(JSON), is_active；含 ToResponse 方法
- [x] 1.2 新增 MCPServer model（ai_mcp_servers 表）：name, description, transport(sse|stdio), url, command, args(JSON), env(JSON), auth_type, auth_config_encrypted(blob), is_active；含 ToResponse 方法（mask auth）
- [x] 1.3 新增 Skill model（ai_skills 表）：name, display_name, description, source_type(github|upload), source_url, manifest(JSON), instructions(text), tools_schema(JSON), auth_type, auth_config_encrypted(blob), is_active；含 ToResponse 方法
- [x] 1.4 新增三张绑定表 model：AgentTool(agent_id, tool_id)、AgentMCPServer(agent_id, mcp_server_id)、AgentSkill(agent_id, skill_id)
- [x] 1.5 在 AI App 的 Models() 中注册所有新表，确保 AutoMigrate

## 2. 内建工具（Tools）后端

- [x] 2.1 新增 ToolRepository：List、GetByID、Update（仅 is_active 字段）
- [x] 2.2 新增 ToolService：ListTools、ToggleTool
- [x] 2.3 新增 ToolHandler：GET /api/v1/ai/tools（列表）、PUT /api/v1/ai/tools/:id（toggle）
- [x] 2.4 在 AI App seed 中注册内建工具：search_knowledge、read_document、http_request（幂等检查）

## 3. MCP 服务后端

- [x] 3.1 新增 MCPServerRepository：CRUD + List with pagination
- [x] 3.2 新增 MCPServerService：Create、Update、Delete、List、GetByID、TestConnection（SSE only）；auth_config 加解密复用 Provider 的 crypto 方法
- [x] 3.3 新增 MCPServerHandler：POST/GET/PUT/DELETE /api/v1/ai/mcp-servers + POST /api/v1/ai/mcp-servers/:id/test

## 4. 技能包（Skills）后端

- [x] 4.1 新增 SkillRepository：CRUD + List with pagination
- [x] 4.2 新增 SkillService：InstallFromGitHub（拉取 manifest+instructions+tools）、InstallFromUpload（解包 tar.gz）、Update（auth_config）、Delete、List、GetByID
- [x] 4.3 新增 SkillHandler：POST /api/v1/ai/skills/import-github（GitHub URL 导入）、POST /api/v1/ai/skills/upload（tar.gz 上传）、GET/PUT/DELETE /api/v1/ai/skills 常规 CRUD
- [x] 4.4 新增内部 API：GET /api/v1/ai/internal/skills/:id/package（Agent 下载，返回 instructions+tools_schema+decrypted auth）

## 5. 绑定表与 soul_config 组装

- [x] 5.1 新增 AgentToolRepository / AgentMCPServerRepository / AgentSkillRepository：绑定与解绑
- [x] 5.2 新增 soul_config 组装逻辑：查询 Agent 绑定的 tools+mcp_servers+skills，组装完整配置（tools 展平、MCP 连接信息解密、Skill 下载 URL+checksum 生成）

## 6. IOC 注册与路由

- [x] 6.1 在 AI App 的 Providers() 中注册所有新 repo/service/handler 到 IOC 容器
- [x] 6.2 在 AI App 的 Routes() 中挂载 tool/mcp/skill 三组路由
- [x] 6.3 在 seed.Sync() 中添加内建工具的幂等同步

## 7. 前端 - 内建工具 Tab

- [x] 7.1 新增工具管理页面框架：三 Tab 布局（内建工具 | MCP 服务 | 技能包）
- [x] 7.2 内建工具 Tab：卡片列表展示 name/description/parameters_schema + enable/disable 开关
- [x] 7.3 API hooks：useTools（列表）、useToggleTool（toggle mutation）

## 8. 前端 - MCP 服务 Tab

- [x] 8.1 MCP 服务 Tab：卡片列表展示 name/transport/url(or command)/auth_type/is_active
- [x] 8.2 MCP 添加/编辑 Sheet：传输方式切换（SSE 配置 / STDIO 配置）、认证配置、测试连接按钮（SSE only）
- [x] 8.3 API hooks：useMCPServers（列表）、useMCPServerMutations（CRUD）、useTestMCPConnection

## 9. 前端 - 技能包 Tab

- [x] 9.1 技能包 Tab：卡片列表展示 name/source_type/tool_count/has_instructions/is_active
- [x] 9.2 GitHub 导入 Sheet：URL 输入 + 扫描结果预览 + 确认导入
- [x] 9.3 上传 Sheet：tar.gz 文件上传 + 解析结果预览
- [x] 9.4 技能包详情/编辑 Sheet：查看 instructions + tools 定义 + 编辑 auth_config
- [x] 9.5 API hooks：useSkills（列表）、useSkillMutations（CRUD + import + upload）

## 10. 前端 - 菜单与国际化

- [x] 10.1 在 AI 模块菜单 seed 中添加"工具注册表"菜单项 + Casbin 策略
- [x] 10.2 添加 i18n 词条：en.json + zh-CN.json（tool registry 相关）
- [x] 10.3 在 AI 模块 module.ts 中注册工具管理页面路由
