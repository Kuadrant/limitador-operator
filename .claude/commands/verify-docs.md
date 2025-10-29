# Verify Documentation Accuracy

Perform a comprehensive review of all repository documentation to ensure it accurately reflects the current codebase state.

## Philosophy: Accuracy Over Completeness

This command focuses on **verifying what IS documented is accurate**, not checking what ISN'T documented.

### What to Check (Accuracy):
- ✅ Documented make targets actually exist
- ✅ Documented constants match actual values
- ✅ YAML examples have valid fields and syntax
- ✅ Tool versions match actual requirements
- ✅ Documented file paths exist
- ✅ Default values match implementation
- ✅ Enum values match API types
- ✅ Code examples are syntactically correct

### What NOT to Check (Completeness):
- ❌ Whether all make targets are documented
- ❌ Whether all API fields have examples
- ❌ Whether new features are documented
- ❌ Whether config directories are all listed
- ❌ Whether all constants are in docs

**Rationale**: Completeness is subjective and context-dependent. Different Claude sessions would have different opinions about what "should" be documented. Focusing on accuracy makes the tool reliable and actionable.

## Objective

Find **inaccuracies** in existing documentation where documented information contradicts the actual implementation.

## Documentation Files to Review

The command will **automatically discover** all documentation files:
- `CLAUDE.md` - Main project documentation and AI context (if exists)
- `README.md` - User-facing README (if exists)
- `RELEASE.md` - Release procedures and current process documentation (if exists)
- `doc/*.md` - All markdown files in the doc/ directory

**Excluded files**:
- `CHANGELOG.md` - Historical record in past tense; not current procedural documentation

**Discovery approach**: Use Glob to find all `.md` files in the repository root and `doc/` directory, then verify each one except CHANGELOG.md.

**Rationale for exclusions**: CHANGELOG.md is a historical archive of past changes (past tense). Verifying it would require checking if old PRs actually did what they claimed, which is archaeological work rather than ensuring current documentation accuracy. RELEASE.md, by contrast, documents the current release process and should stay accurate.

## Generic Verification Patterns

The following verification rules apply to **any discovered documentation file**:

### 1. Make Targets (all files)
- **Extract pattern**: Any mention of `make <target>` or `` `make <target>` ``
- **Verify existence**: Test with `make -n <target>` to confirm it exists in Makefile or make/*.mk
- **Verify syntax**: Command examples show correct flag names and values
- **Verify defaults**: Documented default values for Makefile variables match actual Makefile defaults
- **Detect typos**: If target doesn't exist, suggest similar names from Makefile

### 2. Constants and Default Values (all files)
- **Extract pattern**: Any documented constant names, default values, or configuration values
- **Verify values**: Cross-check against source code (primarily api/v1alpha1/limitador_types.go, pkg/limitador/*.go)
- **Verify existence**: Referenced constants actually exist in the stated files
- **Common sources**:
  - Port numbers → api/v1alpha1/limitador_types.go
  - Default replicas, resources → api/v1alpha1/limitador_types.go
  - Enum values → api/v1alpha1/limitador_types.go
  - Configuration constants → pkg/limitador/*.go

### 3. YAML Examples (all files)
- **Extract pattern**: All YAML code blocks (```yaml ... ```)
- **Verify syntax**: YAML is properly formatted (no stray markers like EOF)
- **Verify fields**: All fields exist in api/v1alpha1/limitador_types.go CRD spec
- **Verify types**: Field types match API (string vs int vs object)
- **Verify enums**: Enum fields use valid values from API type definitions
- **Common API fields to check**:
  - spec.storage.redis.configSecretRef
  - spec.storage.redis-cached.options
  - spec.storage.disk.persistentVolumeClaim
  - spec.listener.http.port / spec.listener.grpc.port
  - spec.resourceRequirements

### 4. Tool Versions (all files)
- **Extract pattern**: References to tool versions (e.g., "operator-sdk version 1.32.0", "kind version v0.22.0")
- **Verify consistency**: Same tool versions across all documentation files
- **Verify accuracy**: Versions match Makefile or build configuration
- **Common tools**: operator-sdk, kind, go, kubectl, kubernetes

### 5. File Paths and Directory References (all files)
- **Extract pattern**: Any mention of file paths (e.g., `config/default/`, `controllers/limitador_controller.go`)
- **Verify existence**: Use Glob or Bash to confirm paths exist
- **Verify links**: Internal markdown links point to existing files
- **Common paths**: config/, controllers/, pkg/, api/v1alpha1/

### 6. Code References and Behavior (all files)
- **Extract pattern**: Descriptions of how code works, function names, reconciliation order
- **Verify accuracy**: Cross-reference with actual implementation
- **Common patterns**:
  - Function names (e.g., reconcileSpec(), reconcilePodLimitsHashAnnotation())
  - Execution order (e.g., reconciliation steps)
  - Environment variables (e.g., LOG_LEVEL, LOG_MODE, RELATED_IMAGE_LIMITADOR)
  - Behavior descriptions match code logic

## Validation Approach

**Step 1: Discover Documentation**
- Use Glob to find: `*.md` in repository root (CLAUDE.md, README.md, RELEASE.md, etc.)
- Use Glob to find: `doc/*.md` for all user-facing documentation
- Filter out: CHANGELOG.md (historical archive, not current documentation)
- Create a list of all markdown files to verify

**Step 2: For Each Documentation File**
1. **Read the file** completely
2. **Extract factual claims** using the patterns above:
   - Make targets mentioned (pattern: `make <target>`)
   - Constants and values (pattern: specific numbers, defaults, variable names)
   - YAML code blocks (pattern: ```yaml ... ```)
   - Tool versions (pattern: "version X.Y.Z")
   - File paths (pattern: paths with .go, .md, directory names)
   - Function/code references (pattern: function names, technical terms)

**Step 3: Verify Each Claim**
- **Make targets**: Test with `make -n <target>`
- **Constants**: Grep in api/v1alpha1/limitador_types.go and pkg/limitador/*.go
- **YAML fields**: Cross-reference with API type definitions
- **Tool versions**: Check Makefile and other doc files for consistency
- **File paths**: Use Glob or Bash to verify existence
- **Code behavior**: Read source code to confirm accuracy

**Step 4: Flag Only Inaccuracies**
Report only when documented information contradicts reality:
- Wrong values
- Typos in names
- Non-existent paths/files
- Invalid YAML fields
- Incorrect defaults
- Broken examples
- Mismatched versions across files

## Output Format

Provide a structured report:

```markdown
## Documentation Verification Report

### ✅ Verified Accurate

List all checked documentation areas that are accurate:
- CLAUDE.md: Important Constants - all values verified correct
- doc/storage.md: RedisCached options table matches API
- doc/development.md: All documented make targets exist

### ⚠️ Inaccuracies Found

Only list actual errors where documented information is wrong:

#### [File name] - [Section]
**Issue**: [Clear description of the inaccuracy]
**Location**: [File:line or section]
**Current docs say**: [Exact quote showing the error]
**Actual implementation**: [What the code/Makefile actually says]
**Suggested fix**: [Specific correction]

### 📊 Summary

- Documentation areas checked: X
- Inaccuracies found: Y
- Severity breakdown:
  - Critical (wrong values/broken examples): Z
  - Minor (typos/outdated descriptions): W

### 🎯 Assessment

If no inaccuracies: "Documentation is accurate and matches implementation."
If issues found: "Found [N] inaccuracies that should be corrected."
```

## Important Guidelines

- **Only flag inaccuracies**: Documented information that contradicts reality
- **Ignore completeness**: Don't flag missing coverage or undocumented features
- **Be precise**: Provide exact quotes and line numbers
- **Verify, don't assume**: Always check source code, don't trust your knowledge
- **Detect typos**: If something doesn't exist but a similar name does, flag as likely typo
- **Focus on facts**: Values, names, paths, syntax - not opinions about documentation style

## Examples of What to Flag

✅ **Flag these (inaccuracies)**:
- "DefaultServiceHTTPPort: 8080" but code says 8081
- "make verify-manifets" but Makefile has "verify-manifests"
- YAML example uses `spec.storage.redis.url` but API field is `configSecretRef`
- "operator-sdk version 1.32.0" in doc/ but "1.30.0" in CLAUDE.md
- Link to `doc/storage.yaml` but file doesn't exist
- "Default: 1000" but code comment says "[default: 500]"

❌ **Don't flag these (completeness/opinions)**:
- Make target exists but not documented in CLAUDE.md
- New API field added but no example in doc/
- Config directory exists but not listed
- Feature works but behavior not fully explained
- Internal constant not mentioned in docs

## Tools to Use

- **Glob**:
  - Discover all markdown files: `*.md` and `doc/*.md`
  - Verify file paths and directories exist
- **Read**:
  - Read discovered documentation files
  - Read source code for verification (api/v1alpha1/limitador_types.go, pkg/limitador/*.go, Makefile)
- **Grep**:
  - Find constants, enum values, defaults in source code
  - Search for function names and patterns
- **Bash**:
  - Test make targets with `make -n <target>`
  - Verify directory existence with `test -d`

## Execution Instructions

1. **Start by discovering all documentation files** using Glob
2. **For each file found**, apply the generic verification patterns
3. **Cross-reference** facts against source code
4. **Generate the comprehensive report** showing what's accurate and what's not

This approach is **future-proof** - it will automatically verify any new documentation files added to the repository without requiring updates to this command.

Start the accuracy verification process now.
