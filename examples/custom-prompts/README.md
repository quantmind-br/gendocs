# Custom Prompts Examples

This directory contains example custom prompt configurations that demonstrate how to tailor Gendocs' AI analysis to different use cases.

## Quick Start

1. **Choose an example** that matches your project type
2. **Copy to your project**:
   ```bash
   # From your project root
   mkdir -p .ai/prompts
   cp /path/to/gendocs/examples/custom-prompts/basic-override.yaml .ai/prompts/
   ```
3. **Customize** the prompts to match your specific needs
4. **Run Gendocs** - your custom prompts will be automatically loaded

## Available Examples

### 1. basic-override.yaml
**Best for:** Getting started with custom prompts

Simple examples of overriding:
- Structure analyzer (focus on architectural patterns)
- Dependency analyzer (security-aware)
- Documenter (clear, professional tone)

**Use this if:** You want to customize a few prompts without overwhelming changes.

### 2. microservices.yaml
**Best for:** Microservices and distributed systems

Comprehensive prompts tailored for:
- Service boundary analysis
- API contract evaluation
- Data flow in distributed systems
- Request tracing and resilience patterns

**Use this if:** Your project uses microservices, service-oriented architecture, or has multiple communicating services.

### 3. enterprise-docs.yaml
**Best for:** Enterprise software with compliance requirements

Enforces enterprise standards:
- Formal documentation style
- Security and compliance focus
- Comprehensive operations guides
- Audit trail requirements

**Use this if:** Your project requires enterprise-grade documentation, compliance certifications, or strict security standards.

## Customization Tips

### Mix and Match
You can combine prompts from different examples:

```bash
# Copy base template
cp examples/custom-prompts/basic-override.yaml .ai/prompts/my-prompts.yaml

# Edit to add specific overrides
vim .ai/prompts/my-prompts.yaml
```

### Partial Overrides
You don't need to override all prompts - only override what you need:

```yaml
# .ai/prompts/custom.yaml
# Only override the documenter, keep other prompts as default
documenter_system_prompt: |
  Your custom documenter prompt here...
```

### Multiple Files
The system loads all `.yaml` and `.yml` files in `.ai/prompts/`:

```
.ai/prompts/
├── analyzers.yaml      # Custom analyzer prompts
├── documentation.yaml  # Custom documenter prompts
└── style-guide.yaml    # Project-specific style rules
```

If the same prompt appears in multiple files, the last one loaded wins.

## Prompt Template Variables

All prompts support Go `text/template` syntax for dynamic content:

```yaml
structure_analyzer_system: |
  Analyzing repository: {{.RepoPath}}
  Project: {{.ProjectName}}
  Language: {{.Language}}
```

Available variables depend on the context - check the system prompts in `prompts/` for examples.

## Testing Your Custom Prompts

1. **Verify Loading**: Run with verbose logging to see which prompts are loaded
   ```bash
   ./gendocs analyze --repo-path . --log-level debug
   ```

2. **Check Output**: Compare analysis results before and after customization

3. **Iterate**: Refine prompts based on the quality of generated documentation

## Troubleshooting

### "Missing required prompts" error
- Ensure you're overriding, not replacing the system prompts
- All required prompts must exist (either in system or project prompts)
- Check prompt names match exactly (case-sensitive)

### Custom prompts not being used
- Verify `.ai/prompts/` directory is in your repository root (not in `.ai/` of Gendocs)
- Check file extension is `.yaml` or `.yml`
- Ensure YAML syntax is valid: `yamllint .ai/prompts/*.yaml`

### Prompts from multiple files conflict
- Check which file is loaded last (alphabetical order)
- Rename files to control load order: `01-base.yaml`, `02-custom.yaml`

## Contributing

Have a useful custom prompt configuration? Consider contributing it:

1. Create a new example file following the existing format
2. Add comprehensive comments explaining the use case
3. Update this README with a description
4. Submit a pull request

## Support

- **Documentation**: See main README.md section on "Custom Prompts"
- **System Prompts Reference**: Check `prompts/` directory for available prompt names
- **Issues**: Report problems at https://github.com/user/gendocs/issues
