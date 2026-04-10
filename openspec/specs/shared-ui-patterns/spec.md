# Capability: shared-ui-patterns

## Purpose
Shared frontend UI patterns, reusable hooks, and cross-cutting UI conventions used across multiple pages.

## Requirements

### Requirement: useListPage hook
The system SHALL provide a `useListPage<T>` hook in `hooks/use-list-page.ts` that encapsulates keyword search state, pagination state, and TanStack Query data fetching for paginated list pages.

The hook SHALL accept `queryKey` (string), `endpoint` (string), and optional `pageSize` (number, default 20).

The hook SHALL return: `keyword`, `setKeyword`, `searchKeyword`, `page`, `setPage`, `pageSize`, `totalPages`, `total`, `items` (T[]), `isLoading`, and `handleSearch` (form event handler).

#### Scenario: Users page uses useListPage
- **WHEN** the users page is rendered
- **THEN** it SHALL use `useListPage<User>` instead of inline state + query logic

#### Scenario: Roles page uses useListPage
- **WHEN** the roles page is rendered
- **THEN** it SHALL use `useListPage<Role>` instead of inline state + query logic

### Requirement: Shared SiteInfo type
The system SHALL define the `SiteInfo` interface (`{ appName: string; hasLogo: boolean }`) in `lib/api.ts` and all consumers (`top-nav.tsx`, `settings/index.tsx`) SHALL import from that single location.

#### Scenario: No duplicate SiteInfo definitions
- **WHEN** searching the codebase for `interface SiteInfo`
- **THEN** only one definition SHALL exist in `lib/api.ts`

### Requirement: Improved empty states
Table empty states SHALL display an icon and descriptive text instead of plain text only. The empty state SHALL include a contextual message guiding the user.

#### Scenario: Users table empty state
- **WHEN** the users query returns zero results
- **THEN** the table SHALL show an icon and "暂无用户" message with muted styling

#### Scenario: Config table empty state
- **WHEN** the config query returns zero results
- **THEN** the table SHALL show an icon and "暂无配置项" message with muted styling

### Requirement: DataTable pagination visual style
The DataTablePagination component SHALL use clean styling without dashed borders.

#### Scenario: Pagination without dashed border
- **WHEN** a paginated table has more than one page
- **THEN** the pagination area SHALL render with `pt-4` top padding and no border
- **AND** background SHALL be transparent (no `bg-muted/10`)
