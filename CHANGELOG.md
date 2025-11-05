# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.0.0]

### Added
- Initial evaluation framework for testing MCP servers with AI agents
- MCP proxy system for recording interactions between agents and servers
  - HTTP/SSE and stdio transport protocol support
  - Real-time call history tracking with timestamps
  - Custom header support for HTTP-based MCP servers
- Flexible agent integration system
  - YAML-based agent configuration with templated command execution
  - Support for Claude Code, OpenAI-compatible agents, and custom agents
  - Virtual home directory support for isolated execution
  - Built-in OpenAI-compatible agent implementation
- Task execution engine with four-phase workflow (Setup, Agent Run, Verify, Cleanup)
  - Task difficulty levels (easy, medium, hard)
  - Inline or file-based script execution
  - Automatic cleanup on failure
  - Task filtering via regex patterns (#19)
  - Glob pattern support for loading multiple tasks
- Comprehensive assertion system
  - Tool call assertions: `toolsUsed`, `requireAny`, `toolsNotUsed`, `minToolCalls`, `maxToolCalls`
  - Resource access assertions: `resourcesRead`, `resourcesNotRead`
  - Prompt assertions: `promptsUsed`, `promptsNotUsed`
  - Behavior assertions: `callOrder`, `noDuplicateCalls`
  - Regex pattern matching for flexible validation
- LLM judge system for automated response evaluation (#25)
  - OpenAI-compatible API integration
  - Two evaluation modes: `exact` and `contains`
  - Structured judgement with failure categorization
  - Deterministic evaluation with seed control
- gevals CLI implementation
  - `gevals eval` command to run evaluations from YAML configuration
  - Output formats: text (colorized) or JSON
  - Verbose mode for detailed progress tracking
  - Automatic result file generation
- Progress tracking and reporting
  - Real-time color-coded task status updates
  - Results summary with pass/fail statistics by difficulty level
  - Detailed failure information with reasons
  - JSON output with complete call history for debugging
- GitHub Action for CI/CD integration (#9685b8b)
  - Multi-platform support (Linux, macOS, Windows)
  - Multi-architecture support (amd64, arm64, arm, 386)
  - Configurable task filtering and pass/fail thresholds
  - Artifact upload for results
  - Automatic binary installation
- Release automation workflows (f98e990)
  - Automated release workflows
  - Pre-release workflow for testing
  - Nightly build workflow
  - Multi-platform binary builds
- Comprehensive Kubernetes MCP server examples (#20)
  - 25+ real-world tasks across difficulty levels
  - Claude Code and OpenAI agent configurations
  - Tasks covering basic operations, debugging, RBAC, networking, and more
- YAML-based configuration system
  - `kind: Eval` - Main evaluation specification
  - `kind: Task` - Individual task definitions
  - `kind: Agent` - Agent behavior specifications
  - Environment variable support
  - Relative path resolution

### Changed

### Deprecated

### Removed

### Fixed

### Security
