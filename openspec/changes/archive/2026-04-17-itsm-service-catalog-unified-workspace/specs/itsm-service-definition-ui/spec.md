## MODIFIED Requirements

### Requirement: 服务定义列表页

系统 SHALL 在路由 `/itsm/services` 提供一体化服务目录工作区（由 `itsm-unified-catalog-workspace` 定义），取代原有的独立表格列表页。原有的表格展示、分页、搜索栏、分类下拉筛选由一体化工作区的 Group Section 导航 + 卡片网格替代。

#### Scenario: 默认展示
- **WHEN** 管理员进入 `/itsm/services` 页面
- **THEN** 系统 SHALL 展示一体化工作区：左侧 Group Section 目录导航面板 + 右侧服务卡片网格，默认选中"全部"，按 root 分组展示

#### Scenario: 按分类筛选
- **WHEN** 管理员在左侧导航面板点击某个 child 目录
- **THEN** 系统 SHALL 在右侧卡片网格中仅展示该目录下的服务

#### Scenario: 点击进入详情
- **WHEN** 管理员点击某个服务卡片
- **THEN** 系统 SHALL 导航到 `/itsm/services/:id` 详情页

### Requirement: 服务定义创建流程

管理员 SHALL 能够从一体化工作区通过 Sheet（侧边抽屉）创建新的服务定义，创建成功后自动跳转到详情页继续配置。

#### Scenario: 打开创建 Sheet
- **WHEN** 管理员点击工作区顶部的"新建服务"按钮或卡片网格末尾的引导卡片
- **THEN** 系统 SHALL 打开 Sheet 表单，包含字段：服务名称（必填）、服务编码（必填）、所属分类（必填，下拉选择 child 目录，若当前已选中某目录则预填）、引擎类型（必填，默认 "smart"）、描述（可选）

#### Scenario: 创建成功跳转
- **WHEN** 管理员填写表单并提交，API 返回成功
- **THEN** 系统 SHALL 关闭 Sheet，显示成功提示，刷新服务列表，并自动导航到新创建服务的详情页 `/itsm/services/:id`

#### Scenario: 编码冲突
- **WHEN** 管理员提交的编码已存在
- **THEN** 系统 SHALL 显示错误提示"服务编码已存在"

## REMOVED Requirements

### Requirement: 关键词搜索
**Reason**: 服务数量通常 < 30，左侧目录导航已提供充分的筛选能力，关键词搜索不再需要
**Migration**: 使用左侧 Group Section 目录导航按分类浏览服务

### Requirement: 按引擎类型筛选
**Reason**: 卡片上引擎类型通过品牌色条和 Badge 直观可辨，无需独立筛选器
**Migration**: 视觉扫描卡片品牌色即可区分

### Requirement: 按状态筛选
**Reason**: 卡片底部状态圆点直观标识启用/停用状态，服务数量少无需筛选
**Migration**: 视觉扫描卡片底部状态圆点

### Requirement: 分页
**Reason**: 卡片网格全量加载（pageSize=100），服务数量少不需要分页
**Migration**: 全量展示，无分页
