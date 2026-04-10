# Capability: theme-system

## Purpose
Defines the visual design system including oklch color variables, glass-morphism effects, typography, interaction transitions, and border/radius conventions.

## Requirements

### Requirement: oklch CSS variable theme
The system SHALL define theme CSS variables using oklch color space in globals.css.

#### Scenario: Theme variables loaded
- **WHEN** the application renders
- **THEN** CSS variables SHALL include --background, --foreground, --primary, --accent, --border, --sidebar, --sidebar-accent, --radius using oklch values

### Requirement: Glass-morphism surfaces
Sidebar and TopNav surfaces SHALL use translucent glass-morphism effect.

#### Scenario: Sidebar glass effect
- **WHEN** the sidebar renders
- **THEN** it SHALL apply `bg-white/80 backdrop-blur-2xl` styling

#### Scenario: TopNav glass effect
- **WHEN** the top navigation renders
- **THEN** it SHALL apply `bg-white/80 backdrop-blur-2xl` styling

### Requirement: Plus Jakarta Sans font
The application SHALL use Plus Jakarta Sans as primary font with CJK fallbacks.

#### Scenario: Font stack applied
- **WHEN** text renders in the application
- **THEN** the font-family SHALL be "Plus Jakarta Sans", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", system-ui, sans-serif

### Requirement: Breathing interaction transitions
All interactive elements SHALL use 200ms transition duration.

#### Scenario: Hover state transition
- **WHEN** a user hovers over an interactive element
- **THEN** the background SHALL transition smoothly with 200ms duration using `bg-black/[0.04]` tint

### Requirement: Border and radius conventions
The application SHALL use subtle oklch borders and 0.75rem base radius.

#### Scenario: Border rendering
- **WHEN** bordered elements render
- **THEN** borders SHALL use `border-black/[0.06]` (near-invisible) styling

#### Scenario: Radius application
- **WHEN** rounded elements render
- **THEN** the base radius SHALL be 0.75rem (12px)
