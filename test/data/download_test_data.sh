#!/bin/bash

# Script to download test data for ReadabiliGo
# This script downloads test files from the ReadabiliPy repository
# These are used for direct comparison with the Python implementation

# Create the output directories if they don't exist
mkdir -p reference/html
mkdir -p reference/expected

# Function to download a file from GitHub
download_github_file() {
    local repo=$1
    local path=$2
    local branch=$3
    local output_file=$4

    echo "Downloading $repo/$path ($branch) to $output_file"
    curl -s -L "https://raw.githubusercontent.com/$repo/$branch/$path" >"$output_file"

    # Check if the download was successful
    if [ $? -eq 0 ] && [ -s "$output_file" ]; then
        echo "Successfully downloaded $output_file"
    else
        echo "Failed to download $repo/$path"
        rm -f "$output_file"
    fi
}

echo "Downloading reference test files from ReadabiliPy repository..."

# Main test files used in ReadabiliPy's own tests
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/addictinginfo.com-1_full_page.html" "master" "reference/html/addictinginfo.html"
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/addictinginfo.com-1_simple_article_from_full_page.json" "master" "reference/expected/addictinginfo.json"

download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/conservativehq.com-1_full_page.html" "master" "reference/html/conservativehq.html"
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/conservativehq.com-1_simple_article_from_full_page.json" "master" "reference/expected/conservativehq.json"

download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/davidwolfe.com-1_full_page.html" "master" "reference/html/davidwolfe.html"
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/davidwolfe.com-1_simple_article_from_full_page.json" "master" "reference/expected/davidwolfe.json"

# Additional test files
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/buzzfeed.com-1_full_page.html" "master" "reference/html/buzzfeed.html"
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/buzzfeed.com-1_simple_article_from_full_page.json" "master" "reference/expected/buzzfeed.json"

download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/econbrowser.com-1_full_page.html" "master" "reference/html/econbrowser.html"
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/econbrowser.com-1_simple_article_from_full_page.json" "master" "reference/expected/econbrowser.json"

download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/en.wikipedia.org-1_full_page.html" "master" "reference/html/wikipedia.html"
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/en.wikipedia.org-1_simple_article_from_full_page.json" "master" "reference/expected/wikipedia.json"

download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/extremetech.com-1_full_page.html" "master" "reference/html/extremetech.html"
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/extremetech.com-1_simple_article_from_full_page.json" "master" "reference/expected/extremetech.json"

download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/nakedcapitalism.com-1_full_page.html" "master" "reference/html/nakedcapitalism.html"
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/nakedcapitalism.com-1_simple_article_from_full_page.json" "master" "reference/expected/nakedcapitalism.json"

# Download the example with edge cases
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/theatlantic.com-1_full_page.html" "master" "reference/html/theatlantic.html"
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/theatlantic.com-1_simple_article_from_full_page.json" "master" "reference/expected/theatlantic.json"

# Download the large file for benchmarking
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/benchmarkinghuge.html" "master" "reference/html/benchmarkinghuge.html"

echo "Download complete. Reference files saved to the reference directory."
echo "- HTML files in reference/html/"
echo "- Expected JSON output in reference/expected/"