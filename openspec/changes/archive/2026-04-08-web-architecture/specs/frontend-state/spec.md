## ADDED Requirements

### Requirement: Fetch wrapper with error handling
The application SHALL provide a typed fetch wrapper in lib/api.ts.

#### Scenario: Successful API call
- **WHEN** a GET request to /api/v1/config returns 200
- **THEN** the wrapper SHALL parse the response JSON and return the data field

#### Scenario: API error response
- **WHEN** an API request returns a non-2xx status
- **THEN** the wrapper SHALL throw an error with the server's error message

### Requirement: TanStack Query for server state
The application SHALL use TanStack Query for all API data fetching with automatic caching and invalidation.

#### Scenario: Query provider setup
- **WHEN** the application mounts
- **THEN** a QueryClientProvider SHALL wrap the entire app

#### Scenario: Data refetch on mutation
- **WHEN** a mutation (create/update/delete) succeeds
- **THEN** related queries SHALL be automatically invalidated and refetched

### Requirement: Zustand for client state
The application SHALL use Zustand for UI-related client state.

#### Scenario: Sidebar state management
- **WHEN** sidebar collapse state changes
- **THEN** it SHALL be managed via a Zustand store and persist across page navigations
