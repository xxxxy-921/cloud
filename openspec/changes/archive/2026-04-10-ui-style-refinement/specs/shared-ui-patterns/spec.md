## MODIFIED Requirements

### Requirement: Improved empty states
Table empty states SHALL display an icon and descriptive text instead of plain text only. The empty state SHALL include a contextual message guiding the user.

#### Scenario: Users table empty state
- **WHEN** the users query returns zero results
- **THEN** the table SHALL show an icon and "暂无用户" message with muted styling

### Requirement: DataTable pagination visual style
The DataTablePagination component SHALL use clean styling without dashed borders.

#### Scenario: Pagination without dashed border
- **WHEN** a paginated table has more than one page
- **THEN** the pagination area SHALL render with `pt-4` top padding and no border
- **AND** background SHALL be transparent (no `bg-muted/10`)
