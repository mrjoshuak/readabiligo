# Edge Case Test Files

This directory contains HTML test files for various edge cases that challenge ReadabiliGo's content extraction capabilities. These tests are designed to verify that the library handles challenging real-world scenarios correctly.

## Test Cases

1. **footer_test.html**
   - Tests proper handling of various footer elements
   - Includes semantic footers, class-based footers, footers with important links
   - Verifies important link preservation behavior

2. **table_layout_test.html**
   - Tests extraction from table-based layouts common in older websites
   - Includes both layout tables (that should be simplified) and data tables (that should be preserved)
   - Tests navigation removal in table-heavy layouts

3. **nested_content_test.html**
   - Tests extraction from deeply nested div structures
   - Contains content with 6+ levels of nesting
   - Verifies important content is still discovered despite excessive nesting

4. **minimal_content_test.html**
   - Tests extraction from pages with minimal content like login pages
   - Verifies form elements are preserved while navigation is removed
   - Tests content-type detection for minimal content

5. **paywall_content_test.html**
   - Tests extraction from articles with content behind paywalls
   - Includes visible content, paywall notification, and premium content
   - Verifies that content behind paywalls is properly extracted

## Issues Identified

The tests revealed several important issues that need to be addressed:

1. **Important Link Preservation**
   - The PreserveImportantLinks option doesn't work correctly
   - "Read more" and "Continue Reading" links aren't preserved as expected

2. **Table Layout Handling**
   - Nested tables aren't properly simplified
   - Navigation elements within tables aren't always removed

3. **Deeply Nested Content Extraction**
   - Content within many levels of divs isn't consistently detected
   - H1 elements and other important content can be lost in deeply nested structures

4. **Metadata Extraction**
   - Date extraction has issues with certain formats
   - Byline extraction can be inconsistent

5. **Content-Type Detection**
   - Minimal content type isn't triggered for obvious login pages
   - Paywall content might need special handling

These test files provide a way to verify improvements as the issues are addressed in future updates.