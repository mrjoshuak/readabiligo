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
    # Download to a temporary file first
    local temp_file="$output_file.tmp"
    curl -s -L "https://raw.githubusercontent.com/$repo/$branch/$path" >"$temp_file"

    # Check if the download was successful
    if [ $? -eq 0 ] && [ -s "$temp_file" ]; then
        # Check if the content contains a 404 error message
        if grep -q "404: Not Found" "$temp_file" || grep -q "404 Not Found" "$temp_file"; then
            echo "⚠️  File $path not found in $branch branch"
            rm -f "$temp_file"
            return 1
        else
            # Move the temporary file to the final location
            mv "$temp_file" "$output_file"
            echo "✅ Successfully downloaded $output_file"
            return 0
        fi
    else
        echo "⚠️  Failed to download $repo/$path"
        rm -f "$temp_file"
        return 1
    fi
}

# Function to attempt download from multiple branches
try_multiple_branches() {
    local repo=$1
    local path=$2
    local output_file=$3
    local branches=("main" "master" "update-tests")
    
    for branch in "${branches[@]}"; do
        if download_github_file "$repo" "$path" "$branch" "$output_file"; then
            return 0
        fi
    done
    
    echo "❌ Could not find $path in any branch"
    return 1
}

echo "Downloading reference test files from ReadabiliPy repository..."

# Counter for successful downloads
success_count=0
total_count=0

# Download addictinginfo files
try_multiple_branches "alan-turing-institute/ReadabiliPy" "tests/data/addictinginfo.com-1_full_page.html" "reference/html/addictinginfo.html"
((total_count++))
[ $? -eq 0 ] && ((success_count++))

try_multiple_branches "alan-turing-institute/ReadabiliPy" "tests/data/addictinginfo.com-1_simple_article_from_full_page.json" "reference/expected/addictinginfo.json"
((total_count++))
[ $? -eq 0 ] && ((success_count++))

# Download conservativehq files
try_multiple_branches "alan-turing-institute/ReadabiliPy" "tests/data/conservativehq.com-1_full_page.html" "reference/html/conservativehq.html"
((total_count++))
[ $? -eq 0 ] && ((success_count++))

try_multiple_branches "alan-turing-institute/ReadabiliPy" "tests/data/conservativehq.com-1_simple_article_from_full_page.json" "reference/expected/conservativehq.json"
((total_count++))
[ $? -eq 0 ] && ((success_count++))

# Download davidwolfe files
try_multiple_branches "alan-turing-institute/ReadabiliPy" "tests/data/davidwolfe.com-1_full_page.html" "reference/html/davidwolfe.html"
((total_count++))
[ $? -eq 0 ] && ((success_count++))

try_multiple_branches "alan-turing-institute/ReadabiliPy" "tests/data/davidwolfe.com-1_simple_article_from_full_page.json" "reference/expected/davidwolfe.json"
((total_count++))
[ $? -eq 0 ] && ((success_count++))

# Download list_items files
try_multiple_branches "alan-turing-institute/ReadabiliPy" "tests/data/list_items_full_page.html" "reference/html/list_items.html"
((total_count++))
[ $? -eq 0 ] && ((success_count++))

try_multiple_branches "alan-turing-institute/ReadabiliPy" "tests/data/list_items_simple_article_from_full_page.json" "reference/expected/list_items.json"
((total_count++))
[ $? -eq 0 ] && ((success_count++))

# Download non_article files
try_multiple_branches "alan-turing-institute/ReadabiliPy" "tests/data/non_article_full_page.html" "reference/html/non_article.html"
((total_count++))
[ $? -eq 0 ] && ((success_count++))

try_multiple_branches "alan-turing-institute/ReadabiliPy" "tests/data/non_article_full_page.json" "reference/expected/non_article.json"
((total_count++))
[ $? -eq 0 ] && ((success_count++))

# Download benchmarking file
try_multiple_branches "alan-turing-institute/ReadabiliPy" "tests/data/benchmarkinghuge.html" "reference/html/benchmarkinghuge.html"
((total_count++))
[ $? -eq 0 ] && ((success_count++))

echo ""
echo "Download summary:"
echo "- Successfully downloaded $success_count out of $total_count files"
echo "- HTML files are in reference/html/"
echo "- Expected JSON output is in reference/expected/"

# Check if we have at least some downloads
if [ $success_count -eq 0 ]; then
    echo "⚠️  WARNING: No files were downloaded successfully. The reference tests will not work."
    exit 1
elif [ $success_count -lt 6 ]; then
    echo "⚠️  WARNING: Only $success_count files were downloaded successfully. Reference tests may be incomplete."
else
    echo "✅ Download complete with $success_count files. Reference files saved to the reference directory."
fi

# Check if we have at least one pair of HTML and JSON files
html_count=$(ls -1 reference/html/*.html 2>/dev/null | wc -l | tr -d ' ')
json_count=$(ls -1 reference/expected/*.json 2>/dev/null | wc -l | tr -d ' ')

if [ "$html_count" -gt 0 ] && [ "$json_count" -gt 0 ]; then
    echo "✅ Found $html_count HTML files and $json_count JSON files"
else
    echo "⚠️  WARNING: Missing HTML or JSON files. HTML: $html_count, JSON: $json_count"
fi