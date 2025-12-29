# Split large analyzer.go file into separate agent files

## Overview

The file internal/agents/analyzer.go has grown to 497 lines and contains three distinct agent types (AnalyzerAgent, DocumenterAgent, AIRulesGeneratorAgent) plus a shared AgentCreator type. This violates the single responsibility principle and makes navigation and testing difficult.

## Rationale

Large files increase cognitive load, make code reviews harder, and often lead to merge conflicts. Each agent type should be in its own file to improve organization, testability, and maintainability.

---
*This spec was created from ideation and is pending detailed specification.*
