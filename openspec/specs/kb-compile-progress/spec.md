# kb-compile-progress Specification

## Purpose
TBD - created by archiving change knowledge-compile-realtime-progress. Update Purpose after archive.
## Requirements
### Requirement: Knowledge base compile progress tracking
The system SHALL track and expose real-time progress of knowledge base compilation, including completed counts for each phase (sources, nodes, embeddings).

#### Scenario: Compile starts and progress is initialized
- **WHEN** a knowledge base compilation starts
- **THEN** the system SHALL initialize compile progress with stage "preparing" and sources total set to the number of sources to process

#### Scenario: Progress updates during source reading
- **WHEN** the system reads each source document during compilation
- **THEN** the system SHALL increment the sources done count
- **AND** update the current item to show which source is being processed

#### Scenario: Progress updates during LLM analysis
- **WHEN** the system calls the LLM for analysis
- **THEN** the system SHALL set stage to "calling_llm"
- **AND** set current item to indicate AI analysis is in progress

#### Scenario: Progress updates when LLM returns node count
- **WHEN** the LLM returns the compiled nodes
- **THEN** the system SHALL set nodes total to the number of nodes to create
- **AND** set embeddings total to the same number

#### Scenario: Progress updates during node writing
- **WHEN** the system writes each node to the knowledge graph
- **THEN** the system SHALL increment the nodes done count
- **AND** update current item to show which node is being created

#### Scenario: Progress updates during embedding generation
- **WHEN** the system generates embeddings for each node
- **THEN** the system SHALL increment the embeddings done count
- **AND** update current item to show which node is being indexed

#### Scenario: Progress marks completion
- **WHEN** the compilation completes successfully
- **THEN** the system SHALL set stage to "completed"
- **AND** set all done counts equal to their totals

### Requirement: Compile progress API
The system SHALL provide an API endpoint to retrieve the current compile progress for a knowledge base.

#### Scenario: Retrieve compile progress
- **WHEN** client sends GET request to `/api/v1/ai/knowledge-bases/{id}/progress`
- **THEN** the system SHALL return the current compile progress including stage, sources count, nodes count, embeddings count, and current item

### Requirement: Frontend progress display
The system SHALL display compile progress in real-time on the knowledge base detail page.

#### Scenario: Progress bar displays current stage
- **WHEN** user views a knowledge base that is compiling
- **THEN** the page SHALL display the current compile stage with a progress indicator

#### Scenario: Progress shows real item counts
- **WHEN** the knowledge base is being compiled
- **THEN** the progress display SHALL show actual done/total counts for sources, nodes, and embeddings
- **AND** update automatically every 2 seconds

#### Scenario: Current activity is shown
- **WHEN** the compilation is processing a specific item
- **THEN** the progress display SHALL show the current item being processed (e.g., "正在创建节点: Claude API")

