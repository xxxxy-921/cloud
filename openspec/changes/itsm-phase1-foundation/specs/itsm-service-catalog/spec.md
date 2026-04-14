## ADDED Requirements

### Requirement: 服务目录树形分类管理

系统 SHALL 支持服务目录的树形分类管理。ServiceCatalog 模型包含 name、description、icon、parent_id（自关联）、sort_order、is_active 字段，嵌入 BaseModel。

#### Scenario: 创建顶级分类
- **WHEN** 管理员创建分类，parent_id 为空
- **THEN** 系统创建顶级分类节点

#### Scenario: 创建子分类
- **WHEN** 管理员创建分类，指定 parent_id 为已有分类 ID
- **THEN** 系统创建该分类的子节点

#### Scenario: 获取分类树
- **WHEN** 请求 GET /api/v1/itsm/catalogs/tree
- **THEN** 系统返回完整的树形结构（递归嵌套 children）

#### Scenario: 删除分类校验
- **WHEN** 管理员删除一个分类，该分类下有子分类或已关联服务定义
- **THEN** 系统拒绝删除并返回错误提示

#### Scenario: 分类排序
- **WHEN** 管理员修改分类的 sort_order
- **THEN** 同层级分类按 sort_order 升序排列

### Requirement: 服务定义管理

系统 SHALL 支持服务定义的 CRUD。ServiceDefinition 模型包含：name、code（唯一）、description、catalog_id（FK→ServiceCatalog）、engine_type（"classic"|"smart"）、sla_id（FK→SLATemplate，可选）、form_schema（JSON，提单表单定义）、workflow_json（JSON，经典模式工作流定义）、collaboration_spec（文本，智能模式协作规范）、agent_id（uint，智能模式关联的 Agent ID）、knowledge_base_ids（JSON 数组，智能模式关联的知识库）、agent_config（JSON，智能模式配置如信心阈值）、is_active、sort_order，嵌入 BaseModel。

#### Scenario: 创建经典服务
- **WHEN** 管理员创建服务定义，engine_type 为 "classic"
- **THEN** 系统保存服务定义，workflow_json 和 form_schema 字段可编辑

#### Scenario: 创建智能服务
- **WHEN** 管理员创建服务定义，engine_type 为 "smart"
- **THEN** 系统保存服务定义，collaboration_spec、agent_id、knowledge_base_ids、agent_config 字段可编辑

#### Scenario: 服务编码唯一性
- **WHEN** 创建服务定义时 code 与已有服务重复
- **THEN** 系统返回 409 错误

#### Scenario: 服务列表分页查询
- **WHEN** 请求 GET /api/v1/itsm/services?page=1&pageSize=10&catalog_id=X
- **THEN** 系统返回该分类下的服务列表（支持按 keyword 搜索）

#### Scenario: 启用/禁用服务
- **WHEN** 管理员修改服务的 is_active 状态
- **THEN** 禁用的服务不出现在用户提单的服务目录中

### Requirement: 动作定义管理

系统 SHALL 支持服务动作（ServiceAction）的 CRUD。ServiceAction 模型包含：name、code（同一服务内唯一）、description、prompt（动作说明）、action_type（"http"）、config_json（JSON，HTTP 配置：url/method/headers/body 模板）、service_id（FK→ServiceDefinition）、is_active，嵌入 BaseModel。

#### Scenario: 创建 HTTP 动作
- **WHEN** 管理员为某服务创建动作，action_type 为 "http"，提供 config_json
- **THEN** 系统保存动作定义，config_json MUST 包含 url 和 method 字段

#### Scenario: 同服务内编码唯一
- **WHEN** 在同一服务下创建 code 重复的动作
- **THEN** 系统返回错误

#### Scenario: 查询服务的动作列表
- **WHEN** 请求 GET /api/v1/itsm/services/:id/actions
- **THEN** 系统返回该服务下的所有动作定义

### Requirement: 优先级管理

系统 SHALL 支持工单优先级定义。Priority 模型包含：name、code（唯一，如 "P0"~"P4"）、value（数字，越小越紧急）、color（十六进制颜色）、description、default_response_minutes、default_resolution_minutes、is_active，嵌入 BaseModel。

#### Scenario: CRUD 优先级
- **WHEN** 管理员创建/编辑/删除优先级
- **THEN** 系统执行相应操作，code 唯一性校验

#### Scenario: 优先级列表排序
- **WHEN** 请求优先级列表
- **THEN** 按 value 升序返回（P0 最紧急排第一）

#### Scenario: Seed 默认优先级
- **WHEN** 系统首次安装或 Sync 启动
- **THEN** 系统创建 P0(紧急)、P1(高)、P2(中)、P3(低)、P4(最低) 五个默认优先级（幂等）

### Requirement: SLA 模板管理

系统 SHALL 支持 SLA 模板定义。SLATemplate 模型包含：name、code（唯一）、description、response_minutes（响应时间）、resolution_minutes（解决时间）、is_active，嵌入 BaseModel。

#### Scenario: CRUD SLA 模板
- **WHEN** 管理员创建/编辑/删除 SLA 模板
- **THEN** 系统执行相应操作

#### Scenario: SLA 绑定到服务
- **WHEN** 管理员编辑服务定义，选择 sla_id
- **THEN** 该服务创建的工单使用此 SLA 的时间要求

### Requirement: 升级规则管理

系统 SHALL 支持 SLA 升级规则。EscalationRule 模型包含：sla_id（FK→SLATemplate）、trigger_type（"response_timeout"|"resolution_timeout"）、level（升级级别 1/2/3）、wait_minutes（等待分钟数）、action_type（"notify"|"reassign"|"escalate_priority"）、target_config（JSON，通知对象/目标处理人/目标优先级）、is_active，嵌入 BaseModel。

#### Scenario: 创建升级规则
- **WHEN** 管理员为某 SLA 创建升级规则
- **THEN** 系统保存规则，同一 SLA + trigger_type 下 level MUST 唯一

#### Scenario: 查询 SLA 的升级规则链
- **WHEN** 请求 GET /api/v1/itsm/sla/:id/escalations
- **THEN** 系统返回该 SLA 的升级规则列表，按 level 升序

### Requirement: 服务目录浏览（用户提单入口）

系统 SHALL 为普通用户提供服务目录浏览页面，用于经典入口提单。

#### Scenario: 浏览启用的服务
- **WHEN** 用户访问 ITSM 提单页面
- **THEN** 系统展示服务目录树，仅显示 is_active=true 的分类和服务

#### Scenario: 搜索服务
- **WHEN** 用户在提单页面搜索关键词
- **THEN** 系统在启用的服务中按名称和描述模糊匹配

### Requirement: ITSM API 路由注册

ITSM App 的所有 API MUST 注册在 `/api/v1/itsm/` 前缀下，并受 JWT + Casbin + Audit 中间件保护。

#### Scenario: 路由前缀
- **WHEN** ITSM App 的 Routes() 被调用
- **THEN** 所有路由挂载在传入的 authedGroup 的 `/itsm` 子组下

#### Scenario: 权限控制
- **WHEN** 未授权用户访问 ITSM API
- **THEN** Casbin 中间件返回 403
