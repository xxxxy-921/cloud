## ADDED Requirements

### Requirement: Message display uses document-style layout
The chat interface SHALL render messages in a document-stream layout without bubble borders.

#### Scenario: User message display
- **WHEN** a user message is rendered
- **THEN** it appears as a block with left border accent
- **AND** the font size is slightly smaller than AI responses
- **AND** it is left-aligned (not right-aligned)

#### Scenario: AI message display
- **WHEN** an AI message is rendered
- **THEN** it appears as plain text without bubble borders
- **AND** it is left-aligned
- **AND** markdown formatting is properly styled

### Requirement: Chat layout maximizes content area
The chat interface SHALL use a single-column layout with maximum reading width.

#### Scenario: Desktop view
- **WHEN** the chat is viewed on desktop
- **THEN** the content area uses max-width of 65ch-75ch
- **AND** the sidebar can be collapsed to expand content area

#### Scenario: Sidebar collapsed
- **WHEN** user clicks the collapse button
- **THEN** the sidebar is hidden
- **AND** the chat content expands to use the freed space

### Requirement: Input area supports multiline with auto-resize
The message input SHALL support multiple lines and automatically adjust height.

#### Scenario: Multiline input
- **WHEN** user types a long message with line breaks
- **THEN** the input area expands vertically up to a max height
- **AND** a scrollbar appears if content exceeds max height

#### Scenario: Send with Enter
- **WHEN** user presses Enter without Shift
- **THEN** the message is sent

#### Scenario: New line with Shift+Enter
- **WHEN** user presses Shift+Enter
- **THEN** a new line is inserted in the input

### Requirement: Message actions are accessible but unobtrusive
Message action buttons SHALL be available but not visually prominent.

#### Scenario: Copy message
- **WHEN** user hovers over or focuses a message
- **THEN** a copy button appears
- **AND** clicking it copies the message content to clipboard

#### Scenario: Regenerate response
- **WHEN** viewing an AI response
- **THEN** a regenerate button is available
- **AND** clicking it triggers response regeneration

#### Scenario: Feedback buttons
- **WHEN** viewing an AI response
- **THEN** thumbs up/down buttons are available
- **AND** clicking them records user feedback

### Requirement: Code blocks have enhanced styling
Code blocks in AI responses SHALL have syntax highlighting and copy functionality.

#### Scenario: Code block display
- **WHEN** a code block is rendered
- **THEN** it uses a dark theme background
- **AND** the programming language is displayed in the top-right
- **AND** a copy button appears on hover

#### Scenario: Copy code
- **WHEN** user clicks the copy button on a code block
- **THEN** the code content is copied to clipboard
- **AND** visual feedback confirms the copy action

### Requirement: Streaming content renders smoothly
Streaming AI responses SHALL render without layout shift or flickering.

#### Scenario: Content streaming
- **WHEN** AI response is streaming in
- **THEN** new content appears smoothly
- **AND** the scroll position is maintained appropriately
- **AND** no layout shift occurs
