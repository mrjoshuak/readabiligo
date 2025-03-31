# Test Data for ReadabiliGo

This directory contains test data for ReadabiliGo. Due to copyright restrictions, some test data files are not included in the repository.

## Test Data Types

There are two main types of test data used:

1. **Reference Data**: Examples from the Python ReadabiliPy repository, used for direct comparison
2. **Real-World Examples**: Current web pages used for performance testing and qualitative evaluation

## Downloading Test Data

Two scripts are provided to download test data:

### 1. Reference Data

The `download_test_data.sh` script downloads reference test cases from the Python ReadabiliPy repository. These are the test cases that the Python implementation is specifically designed to handle correctly.

```bash
# Make the script executable
chmod +x download_test_data.sh

# Run the script
./download_test_data.sh
```

The script downloads:
- HTML files (stored in `reference/html/`)
- Expected JSON output (stored in `reference/expected/`)

Our Go implementation is expected to produce very similar results to the Python version on these specific test cases. This provides a baseline comparison that confirms our implementation follows the same core algorithm.

### 2. Real-World Examples

The `download_real_world_examples.sh` script downloads current web pages for broader testing and benchmarking.

```bash
# Make the script executable
chmod +x download_real_world_examples.sh

# Run the script
./download_real_world_examples.sh
```

**Important Note**: These real-world files may contain copyrighted content and should **NOT** be committed to the repository. The `real_world` directory is included in `.gitignore` to prevent accidental commits.

## Test Files

The test files included in this repository are:

- `list_items_full_page.html`: A test page with list items
- `list_items_simple_article_from_full_page.json`: Expected output for the list items test
- `non_article_full_page.html`: A test page that is not an article
- `non_article_full_page.json`: Expected output for the non-article test

## Running Tests

Once you have downloaded the test data, you can run the tests:

```bash
cd readabiligo
go test ./...
```

For benchmarks:

```bash
cd readabiligo
go test -bench=. ./...
```

## Test Types

1. **Unit Tests**: Tests individual components of the library
2. **Reference Tests (`reference_test.go`)**: Compares our output against Python ReadabiliPy reference cases
3. **Real-World Tests (`comparison_test.go`)**: Tests extraction from current real-world web pages
4. **Benchmarks (`benchmark_test.go`)**: Performance tests
