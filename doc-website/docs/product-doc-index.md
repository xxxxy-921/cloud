---
title: 产品文档索引（自动维护）
unlisted: true
---

# 产品文档索引（自动维护）

最后更新：2026-05-08 21:40 (+08:00)
本轮范围：ITSM

## 已发现结构

### 业务域：系统管理

- 模块：用户管理（/users）
- 模块：角色管理（/roles）
- 模块：菜单管理（/menus）
- 模块：会话管理（/sessions）
- 模块：系统设置（/settings）
- 模块：任务管理（/tasks）
- 模块：公告管理（/announcements）
- 模块：通知渠道（/channels）
- 模块：认证源（/auth-providers）
- 模块：审计日志（/audit-logs）
- 模块：身份源（/identity-sources）

### 业务域：许可管理

- 模块：商品管理（/license/products）
- 模块：授权主体（/license/licensees）
- 模块：注册码管理（/license/registrations）
- 模块：许可签发（/license/licenses）

### 业务域：ITSM

- 模块：服务台（/itsm/service-desk）
- 模块：我的工单（/itsm/tickets/mine）
- 模块：我的待办（/itsm/tickets/approvals/pending）
- 模块：历史工单（/itsm/tickets/approvals/history）
- 模块：服务目录（/itsm/services）
- 模块：优先级管理（/itsm/priorities）
- 模块：SLA 管理（/itsm/sla）
- 模块：工单监控（/itsm/tickets）
- 模块：智能岗位（/itsm/smart-staffing）
- 模块：引擎设置（/itsm/engine-settings）

## 已生成文档

- 系统管理 / [用户管理](./system-management/user-management.md)
- 系统管理 / [角色管理](./system-management/role-management.md)
- 系统管理 / [菜单管理](./system-management/menu-management.md)
- 系统管理 / [会话管理](./system-management/session-management.md)
- 系统管理 / [系统设置](./system-management/system-settings.md)
- 系统管理 / [任务管理](./system-management/task-management.md)
- 系统管理 / [公告管理](./system-management/announcement-management.md)
- 系统管理 / [通知渠道](./system-management/channel-management.md)
- 系统管理 / [认证源](./system-management/auth-provider-management.md)
- 系统管理 / [身份源管理](./system-management/identity-source-management.md)
- 系统管理 / [审计日志](./system-management/audit-log-management.md)
- 许可管理 / [商品管理](./license-management/product-management.md)
- 许可管理 / [授权主体](./license-management/licensee-management.md)
- 许可管理 / [许可签发](./license-management/license-issuance.md)
- ITSM / [服务目录](./itsm-management/service-catalog-management.md)（本轮新增）
- ITSM / [工单监控](./itsm-management/ticket-monitoring.md)（本轮新增）
- ITSM / [引擎设置](./itsm-management/engine-settings.md)（本轮新增）

## 待补齐（按优先级）

1. 许可管理 / 注册码管理（/license/registrations）
   - 原因：已发现用户入口与权限点（`license:registration:list`），尚无独立用户文档。
   - 优先级：中（对签发效率有影响，但数据影响低于签发与吊销）。
2. ITSM / 服务台（/itsm/service-desk）
   - 原因：已发现独立菜单入口，但当前尚无正式用户文档。
   - 优先级：中（与用户提交工单直接相关）。
3. ITSM / 我的工单（/itsm/tickets/mine）
   - 原因：已有明确工作台入口和筛选行为，当前尚无独立页面说明。
   - 优先级：中（高频个人入口）。
4. ITSM / 我的待办（/itsm/tickets/approvals/pending）
   - 原因：审批入口清晰，但尚无正式用户文档。
   - 优先级：中（高频审批入口）。
5. ITSM / 历史工单（/itsm/tickets/approvals/history）
   - 原因：已有独立菜单入口，但与待办相比紧迫度更低。
   - 优先级：低。
6. ITSM / 优先级管理（/itsm/priorities）
   - 原因：已有完整页面和本地化文案，仍缺正式用户文档。
   - 优先级：中（工单排序与 SLA 协作依赖该配置）。
7. ITSM / SLA 管理（/itsm/sla）
   - 原因：已有独立页面和规则概览，但尚无正式用户文档。
   - 优先级：中（直接影响响应和解决承诺）。
8. ITSM / 智能岗位（/itsm/smart-staffing）
   - 原因：与引擎设置相邻，但当前还缺正式说明。
   - 优先级：中（影响 Agentic ITSM 运行分工）。

## 本轮处理说明

- 匹配结论：ITSM 域长期未触达，且 `服务目录 / 工单监控 / 引擎设置` 同时存在缺页与高价值入口，符合反饥饿调度选择。
- 本轮策略：限定在 ITSM 域，优先补齐入口最清晰、任务链路最完整的 3 个缺页模块。
- 本轮新增：服务目录、工单监控、引擎设置（共 3 篇）。
- 证据对照：
  - 路由：`web/src/apps/itsm/module.ts`
  - 菜单与权限：`internal/app/itsm/bootstrap/seed.go`
  - 页面行为：`web/src/apps/itsm/pages/services/**`、`web/src/apps/itsm/pages/tickets/**`、`web/src/apps/itsm/pages/engine-settings/index.tsx`
  - 中文文案：`web/src/apps/itsm/locales/zh-CN.json`
- 覆盖状态：系统管理 11/11 已覆盖；许可管理 3/4 已覆盖；ITSM 3/10 已覆盖。
