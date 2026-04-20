## 1. Agent And Session Ownership Enforcement

- [ ] 1.1 Add ownership/visibility-aware agent lookup methods in the AI agent repository/service layer for detail, update, delete, and session creation checks.
- [ ] 1.2 Update `AgentHandler` to use ownership-aware agent access for `GET /:id`, `PUT /:id`, and `DELETE /:id`, preserving existing list semantics.
- [ ] 1.3 Update `SessionService` and `SessionHandler` so create/detail/update/delete/send/edit/cancel/continue/upload-image flows always load sessions through current-user ownership checks.
- [ ] 1.4 Update `AgentGateway` to resolve sessions through ownership-aware access before loading history, memories, or starting/cancelling execution.

## 2. History And Memory Isolation Hardening

- [ ] 2.1 Fix memory handlers to read the standard JWT context key (`userId`) and fail safely when the user context is missing.
- [ ] 2.2 Add memory deletion checks so users can only delete memories belonging to their own `(agent_id, user_id)` scope.
- [ ] 2.3 Ensure cross-user session history and message-edit paths return the same not-found behavior as missing sessions.
- [ ] 2.4 Add or update tests covering hidden private agents, cross-user session access, cross-user stream access, and cross-user memory deletion.

## 3. Stream Protocol Encoder Abstraction

- [ ] 3.1 Introduce a protocol-agnostic stream encoder boundary for Gateway event output while keeping the existing Vercel UI stream behavior as the default implementation.
- [ ] 3.2 Refactor current Vercel UI stream encoding code to implement the new encoder boundary without changing `/api/v1/ai/sessions/:sid/stream` semantics.
- [ ] 3.3 Add tests that prove the default encoder remains compatible with the current frontend stream consumer and that Gateway orchestration no longer embeds protocol-specific line construction directly.
