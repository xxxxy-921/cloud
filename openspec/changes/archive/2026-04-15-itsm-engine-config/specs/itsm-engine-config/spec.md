## ADDED Requirements

### Requirement: ITSM 引擎配置聚合 API

系统 SHALL 提供聚合 API `GET /api/v1/itsm/engine/config` 和 `PUT /api/v1/itsm/engine/config`，统一读写 ITSM 智能引擎的全部配置。API 受 JWT + Casbin 权限保护。

响应/请求结构：
```json
{
  "generator": {
    "model_id": 1,
    "provider_id": 1,
    "provider_name": "DeepSeek",
    "model_name": "deepseek-v3",
    "temperature": 0.3
  },
  "runtime": {
    "model_id": 2,
    "provider_id": 1,
    "provider_name": "DeepSeek",
    "model_name": "deepseek-r1",
    "temperature": 0.1,
    "decision_mode": "direct_first"
  },
  "general": {
    "max_retries": 3,
    "timeout_seconds": 30,
    "reasoning_log": "full"
  }
}
```

#### Scenario: 读取引擎配置
- **WHEN** 管理员调用 `GET /api/v1/itsm/engine/config`
- **THEN** 系统 SHALL 读取 `itsm.generator` 和 `itsm.runtime` 两个 internal Agent 的配置（model_id 关联的 provider_id、provider_name、model_name、temperature），以及 SystemConfig 中 `itsm.engine.*` 前缀的运维参数，合并为统一的 JSON 结构返回

#### Scenario: 保存引擎配置
- **WHEN** 管理员调用 `PUT /api/v1/itsm/engine/config` 提交完整配置
- **THEN** 系统 SHALL 更新 `itsm.generator` Agent 的 model_id 和 temperature，更新 `itsm.runtime` Agent 的 model_id 和 temperature，更新 SystemConfig 中的 `itsm.engine.runtime.decision_mode`、`itsm.engine.general.max_retries`、`itsm.engine.general.timeout_seconds`、`itsm.engine.general.reasoning_log`

#### Scenario: Agent 未绑定模型时读取
- **WHEN** 读取配置且 `itsm.generator` Agent 的 model_id 为空（0 或 null）
- **THEN** 系统 SHALL 在 generator 区块返回 model_id=0、provider_id=0、provider_name=""、model_name=""，前端据此展示未配置状态

#### Scenario: 无效 model_id
- **WHEN** 保存配置时提交的 model_id 对应的 AIModel 不存在或已停用
- **THEN** 系统 SHALL 返回 400 错误 "模型不存在或已停用"

### Requirement: ITSM 引擎配置前端页面

系统 SHALL 在 ITSM 模块侧边栏提供「引擎配置」菜单项（路由 `/itsm/engine-config`），页面展示三个配置区块：工作流解析引擎、运行时决策引擎、通用设置。

#### Scenario: 页面加载
- **WHEN** 管理员进入 `/itsm/engine-config` 页面
- **THEN** 系统 SHALL 调用 `GET /api/v1/itsm/engine/config` 加载配置，并调用 `GET /api/v1/ai/providers` 加载 Provider 列表填充下拉框

#### Scenario: Provider-Model 联动
- **WHEN** 管理员在某个引擎区块选择 AI 服务商（Provider）
- **THEN** 系统 SHALL 调用 `GET /api/v1/ai/providers/:id/models`（或等效 API）加载该 Provider 下的模型列表，填充模型下拉框。若之前已选的模型属于新 Provider 则保留，否则清空模型选择

#### Scenario: 保存配置
- **WHEN** 管理员修改配置并点击「保存配置」
- **THEN** 系统 SHALL 调用 `PUT /api/v1/itsm/engine/config` 提交全部配置，成功后显示成功提示

#### Scenario: 未配置 Provider 引导
- **WHEN** 页面加载时 Provider 列表为空（系统未添加任何 AI 服务商）
- **THEN** 系统 SHALL 展示引导提示："请先在 AI 模块添加服务商"，并提供跳转链接

#### Scenario: 决策模式选择
- **WHEN** 管理员在运行时决策引擎区块选择决策模式
- **THEN** 系统 SHALL 提供两个选项：「优先确定路径，回退 AI」（direct_first）和「始终使用 AI 决策」（ai_only），使用 Radio 组件

#### Scenario: 推理日志级别选择
- **WHEN** 管理员在通用设置区块选择推理日志级别
- **THEN** 系统 SHALL 提供三个选项：「完整推理记录」（full）、「仅摘要」（summary）、「关闭」（off），使用 Select 组件

### Requirement: ITSM Seed 引擎默认配置

ITSM App 的 Seed SHALL 在首次安装和后续启动时确保引擎相关的 Agent 记录和 SystemConfig 默认值存在。

#### Scenario: 首次安装创建 internal Agent
- **WHEN** ITSM Seed 运行且数据库中不存在 code 为 `itsm.generator` 的 Agent
- **THEN** Seed SHALL 创建 Agent 记录：code=`itsm.generator`、name="ITSM 工作流解析"、type=`internal`、temperature=0.3、system_prompt=内置解析提示词、model_id=0（未绑定）

#### Scenario: 首次安装创建运行时 Agent
- **WHEN** ITSM Seed 运行且数据库中不存在 code 为 `itsm.runtime` 的 Agent
- **THEN** Seed SHALL 创建 Agent 记录：code=`itsm.runtime`、name="ITSM 运行时决策"、type=`internal`、temperature=0.1、system_prompt=内置决策提示词、model_id=0（未绑定）

#### Scenario: 首次安装写入 SystemConfig 默认值
- **WHEN** ITSM Seed 运行且 SystemConfig 中不存在 key `itsm.engine.runtime.decision_mode`
- **THEN** Seed SHALL 写入以下默认值：`itsm.engine.runtime.decision_mode`=`direct_first`、`itsm.engine.general.max_retries`=`3`、`itsm.engine.general.timeout_seconds`=`30`、`itsm.engine.general.reasoning_log`=`full`

#### Scenario: 后续启动不覆盖已有配置
- **WHEN** ITSM Seed 运行且 Agent 和 SystemConfig 已存在
- **THEN** Seed SHALL 跳过已存在的记录，不覆盖用户自定义的配置值

### Requirement: ITSM 引擎配置菜单注册

ITSM App SHALL 在 Seed 阶段注册「引擎配置」菜单项和对应的权限策略。

#### Scenario: 菜单注册
- **WHEN** ITSM Seed 运行
- **THEN** Seed SHALL 在 ITSM 菜单组下添加「引擎配置」菜单项，路由为 `/itsm/engine-config`，排序在现有菜单末尾

#### Scenario: 权限策略注册
- **WHEN** ITSM Seed 运行
- **THEN** Seed SHALL 通过 Casbin 注册 `GET /api/v1/itsm/engine/config` 和 `PUT /api/v1/itsm/engine/config` 的权限策略，关联到 ITSM 管理角色
