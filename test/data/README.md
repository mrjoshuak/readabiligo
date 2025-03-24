# Test Data for ReadabiliGo

This directory contains test data for ReadabiliGo. Due to copyright restrictions, some test data files are not included in the repository.

## Real-World Examples

The `download_real_world_examples.sh` script can be used to download real-world HTML examples for testing. This script uses curl to download HTML from various websites and saves them to the `real_world` directory.

**Important Note**: The downloaded real-world HTML files may contain copyrighted content and should **NOT** be committed to the repository. The `real_world` directory is included in `.gitignore` to prevent accidental commits.

To download the test data:

```bash
# Make the script executable
chmod +x download_real_world_examples.sh

# Run the script
./download_real_world_examples.sh
```

## Test Files

The test files included in this repository are:

- `list_items_full_page.html`: A test page with list items
- `list_items_simple_article_from_full_page.json`: Expected output for the list items test
- `non_article_full_page.html`: A test page that is not an article
- `non_article_full_page.json`: Expected output for the non-article test

## Additional Test Files

The `download_real_world_examples.sh` script will automatically download the following test files from the ReadabiliPy repository:

- `addictinginfo.com-1_full_page.html`
- `addictinginfo.com-1_simple_article_from_full_page.json`
- `conservativehq.com-1_full_page.html`
- `conservativehq.com-1_simple_article_from_full_page.json`
- `davidwolfe.com-1_full_page.html`
- `davidwolfe.com-1_simple_article_from_full_page.json`
- `benchmarkinghuge.html`

**Important Note**: These files may contain copyrighted content and are excluded from the repository via `.gitignore`. They will be downloaded when you run the script.

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
