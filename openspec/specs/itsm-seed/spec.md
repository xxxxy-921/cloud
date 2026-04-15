## MODIFIED Requirements

### Requirement: ITSM seed includes built-in form definitions
The ITSM seed SHALL create 3 built-in FormDefinition records before creating ServiceDefinition records. Seed SHALL use idempotent creation (check by code before insert). Built-in forms:
1. `form_general_incident` — "通用事件表单": title(text), description(textarea), urgency(select: 低/中/高/紧急), impact(select: 个人/部门/全公司), contact(text)
2. `form_change_request` — "变更申请表单": title(text), description(textarea), reason(textarea), planned_time(datetime), impact_assessment(textarea), rollback_plan(textarea)
3. `form_service_request` — "服务请求表单": title(text), description(textarea), expected_date(date), remarks(textarea)

#### Scenario: First-time seed creates forms
- **WHEN** the ITSM seed runs for the first time (no existing form definitions)
- **THEN** 3 FormDefinition records SHALL be created with version=1, scope="global", is_active=true

#### Scenario: Idempotent re-seed
- **WHEN** the ITSM seed runs again and form definitions already exist
- **THEN** existing form definitions SHALL NOT be overwritten or duplicated

### Requirement: ServiceDefinition seed references forms
The existing seeded ServiceDefinition records SHALL reference the built-in FormDefinition records via `form_id` instead of inline `form_schema`. The seed SHALL look up FormDefinition by code to resolve the ID.

#### Scenario: Service references form
- **WHEN** the seed creates the "Copilot 账号申请" service definition
- **THEN** its `form_id` SHALL reference the `form_service_request` FormDefinition

### Requirement: Menu seed includes form management
The ITSM seed SHALL add a "表单管理" menu item under the ITSM menu group with permission `itsm:form:list` and appropriate Casbin policies for CRUD operations.

#### Scenario: Form menu created
- **WHEN** the ITSM seed runs
- **THEN** a menu item "表单管理" SHALL exist under ITSM with route `/itsm/forms` and permission `itsm:form:list`

#### Scenario: Form CRUD policies
- **WHEN** the ITSM seed runs
- **THEN** Casbin policies SHALL allow admin role to access `POST/GET/PUT/DELETE /api/v1/itsm/forms*`
