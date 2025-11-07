# Refactoring Plan for internal/readability/cleanup.go

## Current State

**File:** `internal/readability/cleanup.go`
**Lines:** 1,412 lines
**Status:** Functional but overly complex
**Priority:** High (maintainability and testability concerns)

## Problem Statement

The cleanup.go file has grown to 1,412 lines and contains deeply nested logic that makes it difficult to:
- Understand the flow of content cleaning
- Test individual components in isolation
- Modify without risking regressions
- Onboard new contributors to the codebase

## Proposed Solution

Split cleanup.go into focused, single-responsibility modules while maintaining 100% backward compatibility and Mozilla Readability.js algorithm adherence.

## File Structure (Proposed)

```
internal/readability/
├── cleanup/
│   ├── cleanup.go              # Main orchestration (200 lines)
│   ├── footer_handler.go       # Footer detection and removal (200 lines)
│   ├── table_classifier.go     # Table classification logic (250 lines)
│   ├── link_preservation.go    # Important link detection (200 lines)
│   ├── conditional_cleaning.go # Conditional node removal (300 lines)
│   ├── metrics.go              # Node metrics calculation (150 lines)
│   └── element_filters.go      # Element filtering utilities (150 lines)
└── cleanup.go (deprecated)     # Kept temporarily for compatibility
```

## Detailed Breakdown

### 1. cleanup.go (Main Orchestration)
**Lines:** ~200
**Responsibility:** High-level cleaning orchestration

**Functions to keep:**
- `Clean()` - Main entry point
- `cleanConditionally()` - Conditional cleaning orchestration
- `cleanTag()` - Tag-based cleaning

**Key characteristics:**
- Delegates to specialized handlers
- Maintains cleaning order
- Coordinates between modules

### 2. footer_handler.go
**Lines:** ~200
**Responsibility:** Footer and aside element handling

**Functions to extract:**
- `cleanElementsFromOriginalDocument()` (lines 65-98)
- `cleanElementsInArticle()` (lines 41-63)
- `preserveImportantLinksIfNeeded()` (lines 99-139)
- Footer-specific logic from conditional cleaning

**Rationale:** Footer handling is complex and self-contained

### 3. table_classifier.go
**Lines:** ~250
**Responsibility:** Table classification and cleaning

**Functions to extract:**
- `isDataTable()` (lines 1001-1119)
- `markDataTables()` (lines 1222-1233)
- `cleanTables()` (lines 956-999)
- Helper functions for table analysis

**Rationale:** Table logic is substantial and rarely changes

### 4. link_preservation.go
**Lines:** ~200
**Responsibility:** Important link detection and preservation

**Functions to extract:**
- `findAndExtractImportantLinks()` (lines 765-796)
- `hasImportantLinks()` (lines 798-808)
- `preserveImportantLinksAnywhere()` (lines 810-887)
- `isImportantLink()` (lines 889-933)
- `extractLinkContext()` (lines 935-954)

**Rationale:** Currently scattered, needs consolidation

### 5. conditional_cleaning.go
**Lines:** ~300
**Responsibility:** Conditional node removal logic

**Functions to extract:**
- `shouldRemoveNode()` (lines 352-387)
- `evaluateRemovalCriteria()` (lines 516-681)
- `shouldSkipConditionalCleaning()` (lines 260-276)
- `shouldPreserveStructure()` (lines 278-350)

**Rationale:** Core algorithm logic needs clear separation

### 6. metrics.go
**Lines:** ~150
**Responsibility:** Node metrics calculation

**Functions to extract:**
- `calculateNodeMetrics()` (lines 430-514)
- `NodeMetrics` struct (lines 389-401)
- Related metric calculation helpers

**Rationale:** Self-contained calculation logic

### 7. element_filters.go
**Lines:** ~150
**Responsibility:** Element filtering utilities

**Functions to extract:**
- `cleanHeaders()` (lines 552-628)
- `isAllowedVideo()` (lines 683-711)
- `shouldKeepImage()` (lines 713-763)
- Other element-specific filters

**Rationale:** Reusable filtering utilities

## Migration Strategy

### Phase 1: Preparation (1-2 days)
1. Create `/internal/readability/cleanup/` package
2. Add comprehensive tests for existing cleanup.go functionality
3. Create baseline benchmarks
4. Document current behavior edge cases

### Phase 2: Extraction (3-5 days)
1. Extract metrics.go first (least dependencies)
2. Extract element_filters.go
3. Extract table_classifier.go
4. Extract link_preservation.go
5. Extract footer_handler.go
6. Extract conditional_cleaning.go
7. Create new cleanup.go orchestrator

### Phase 3: Integration (2-3 days)
1. Update import paths throughout codebase
2. Run full test suite
3. Run benchmarks to verify no performance regression
4. Update documentation

### Phase 4: Validation (1-2 days)
1. Test against real-world HTML corpus
2. Compare output with Mozilla Readability.js
3. Verify no regressions in edge cases
4. Performance profiling

### Phase 5: Cleanup (1 day)
1. Remove old cleanup.go (or mark deprecated)
2. Update CHANGELOG.md
3. Update architecture documentation
4. Code review and merge

**Total estimated time:** 8-13 days

## Testing Strategy

### Unit Tests
- Each extracted file gets its own test file
- Aim for 80%+ code coverage per file
- Test edge cases explicitly

### Integration Tests
- Test orchestration flow in cleanup.go
- Verify interaction between modules
- Test error handling and edge cases

### Regression Tests
- Compare output before/after refactoring
- Use existing test corpus
- Run against Mozilla's test suite if available

### Performance Tests
- Benchmark each module
- Benchmark overall extraction
- Ensure no >5% performance degradation

## Risks and Mitigation

### Risk 1: Breaking Changes
**Likelihood:** Medium
**Impact:** High
**Mitigation:**
- Maintain 100% backward compatibility
- Comprehensive test coverage before refactoring
- Keep old cleanup.go as fallback initially

### Risk 2: Performance Regression
**Likelihood:** Low
**Impact:** Medium
**Mitigation:**
- Benchmark before and after
- Profile hot paths
- Use interface indirection sparingly

### Risk 3: Incorrect Behavior Changes
**Likelihood:** Medium
**Impact:** High
**Mitigation:**
- Test against large HTML corpus
- Compare with Mozilla's output
- Peer code review
- Phased rollout

### Risk 4: Increased Complexity
**Likelihood:** Low
**Impact:** Medium
**Mitigation:**
- Clear module boundaries
- Good documentation
- Simple public APIs
- Minimize cross-module dependencies

## Success Criteria

✅ **All tests pass** - 100% of existing tests still pass
✅ **No performance regression** - <5% difference in benchmarks
✅ **Improved maintainability** - Files <400 lines each
✅ **Better testability** - Can test modules in isolation
✅ **Clear responsibility** - Each file has single, clear purpose
✅ **Documentation** - All public functions documented
✅ **Code coverage** - 80%+ coverage on new files

## Future Improvements (Post-Refactoring)

1. **Link Preservation Refactor**
   - Create `LinkPreserver` struct with strategy pattern
   - Separate detection from preservation logic
   - Add configurable link patterns

2. **Table Classifier Enhancement**
   - Machine learning-based classification (optional)
   - More sophisticated layout vs data detection
   - Support for complex table structures

3. **Metrics Caching**
   - Cache node metrics to avoid recalculation
   - Implement cache invalidation strategy
   - Measure performance improvement

4. **Structured Logging**
   - Replace fmt.Printf with structured logger
   - Add log levels (DEBUG, INFO, WARN, ERROR)
   - Make logging injectable and testable

## Dependencies

**External:**
- None (pure internal refactoring)

**Internal:**
- All changes contained within `internal/readability/`
- No public API changes
- No breaking changes to public interfaces

## Rollback Plan

If critical issues are discovered post-refactoring:

1. **Immediate:** Revert to previous commit (preserve in git history)
2. **Short-term:** Keep old cleanup.go as fallback with feature flag
3. **Long-term:** Fix issues and re-deploy improved version

## References

- Mozilla Readability.js: https://github.com/mozilla/readability
- Go Code Review Comments: https://go.dev/wiki/CodeReviewComments
- Effective Go: https://go.dev/doc/effective_go
- Clean Architecture principles

## Approval Required From

- [ ] Codebase maintainer
- [ ] Technical lead (if applicable)
- [ ] QA team for test plan review

## Status

**Status:** PROPOSED (Not yet started)
**Created:** 2025-01-XX
**Priority:** High
**Complexity:** High
**Estimated Effort:** 8-13 days
**Dependencies:** None
**Blocker:** None

---

*This is a living document. Update as the refactoring progresses.*
