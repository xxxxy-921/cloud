## 1. Message Component Redesign

- [x] 1.1 Create new `MessageItem` component with document-style layout
- [x] 1.2 Implement user message "quote" style with left border accent
- [x] 1.3 Implement AI message plain text rendering without bubbles
- [x] 1.4 Update streaming message display to match new style
- [x] 1.5 Update tool_call and tool_result message styling
- [x] 1.6 Remove or update `MessageBubble` component references

## 2. Chat Layout Restructuring

- [x] 2.1 Update `[sid].tsx` layout to use single-column content flow
- [x] 2.2 Increase content max-width from max-w-3xl to max-w-4xl or 65ch
- [x] 2.3 Remove right/left alignment logic, use left-aligned for all messages
- [x] 2.4 Add proper spacing between messages (space-y-6 or similar)
- [x] 2.5 Update header styling to be more minimal

## 3. Sidebar Collapsible Redesign

- [x] 3.1 Add collapse/expand button to sidebar
- [x] 3.2 Implement sidebar collapsed state with localStorage persistence
- [x] 3.3 Animate sidebar collapse/expand transition
- [x] 3.4 Ensure chat content expands when sidebar is collapsed
- [x] 3.5 Handle mobile view - sidebar as sheet/drawer

## 4. Input Area Enhancement

- [x] 4.1 Convert Textarea to auto-resize with min/max rows
- [x] 4.2 Move input area to floating card style at bottom
- [x] 4.3 Update Send button styling to be more prominent
- [x] 4.4 Ensure proper keyboard handling (Enter to send, Shift+Enter for newline)
- [x] 4.5 Add visual separator between chat content and input area

## 5. Message Actions Implementation

- [x] 5.1 Add copy message button (appears on hover/focus)
- [x] 5.2 Add regenerate response button for AI messages
- [x] 5.3 Add thumbs up/down feedback buttons
- [x] 5.4 Implement copy feedback toast notification
- [x] 5.5 Position actions at message bottom or as floating toolbar

## 6. Code Block Enhancement

- [x] 6.1 Update code block styling to use dark theme
- [x] 6.2 Add language label to code block header
- [x] 6.3 Add copy button to code blocks (appears on hover)
- [x] 6.4 Ensure proper syntax highlighting colors
- [x] 6.5 Handle long code blocks with max-height and overflow

## 7. Markdown Content Styling

- [x] 7.1 Style headings (H1, H2, H3) with proper hierarchy
- [x] 7.2 Update list styling (ul, ol) with proper indentation
- [x] 7.3 Style blockquotes with left border
- [x] 7.4 Update table styling with proper borders and zebra striping
- [x] 7.5 Style inline code with subtle background

## 8. Streaming Experience Polish

- [x] 8.1 Ensure smooth content insertion without layout shift
- [x] 8.2 Add subtle fade-in animation for new content
- [x] 8.3 Handle auto-scroll behavior properly
- [x] 8.4 Remove streaming pulse animation or make it subtler
- [x] 8.5 Test streaming with various content types

## 9. Dark Mode Support

- [x] 9.1 Ensure all new colors use CSS variables for dark mode
- [x] 9.2 Test code block dark theme in dark mode
- [x] 9.3 Verify user message accent border color in dark mode
- [x] 9.4 Test input area styling in dark mode

## 10. Testing and Refinement

- [x] 10.1 Test with long conversations (scrolling performance)
- [x] 10.2 Test with various message types (text, code, markdown, tools)
- [x] 10.3 Test responsive behavior on different screen sizes
- [x] 10.4 Verify keyboard navigation accessibility
- [x] 10.5 Run ESLint and fix any issues
