# Gotchas & Pitfalls

Things to watch out for in this codebase.

## [2025-12-29 04:20]
Import cleanup after code removal: When extracting code from a Go file to new files, remember to remove imports that become unused in the original file. For example, removing DocumenterAgent and AIRulesGeneratorAgent from analyzer.go required removing the 'os' import that was only used by those agents. Unused imports cause compilation errors.

_Context: Go file refactoring: Phase 3-2 and 4-2 of the analyzer.go splitting required removing the 'os' import after extracting agent code to separate files._

## [2025-12-29 04:20]
Export verification is critical for zero-breaking-change refactoring: Before splitting files, verify all exported types (capitalized names) are properly exported and accessible. In Go, types in the same package are automatically accessible regardless of which file they're in, but external consumers depend on exports. After refactoring, verify handler packages still have access to all constructors (NewAnalyzerAgent, NewDocumenterAgent, NewAIRulesGeneratorAgent) and types they use.

_Context: During the analyzer.go refactoring, all agent types and constructors were already properly exported, so no changes were needed for external consumers. Phase 6-2 verified package-level exports were maintained._
