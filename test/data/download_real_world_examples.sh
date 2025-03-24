#!/bin/bash

# Script to download real-world HTML examples for testing ReadabiliGo
# This script uses curl to download HTML from various websites and saves them to the real_world directory
# It also downloads test files from the ReadabiliPy repository

# Create the output directory if it doesn't exist
mkdir -p real_world

# Function to download a URL and save it with a clean filename
download_url() {
	local url=$1
	local filename=$2

	echo "Downloading $url to real_world/$filename.html"
	curl -s -L -A "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36" \
		"$url" >"real_world/$filename.html"

	# Check if the download was successful
	if [ $? -eq 0 ] && [ -s "real_world/$filename.html" ]; then
		echo "Successfully downloaded $filename.html"
	else
		echo "Failed to download $url"
		rm -f "real_world/$filename.html"
	fi
}

# News websites
download_url "https://www.bbc.com/news/world-us-canada-68651895" "bbc_news"
download_url "https://www.nytimes.com/2025/03/23/world/europe/ukraine-russia-war.html" "nytimes"
download_url "https://www.theguardian.com/world/2025/mar/23/global-climate-report" "guardian"
download_url "https://www.reuters.com/world/europe/latest-developments-ukraine-2025-03-23" "reuters"

# Tech blogs and documentation
download_url "https://go.dev/blog/go1.22" "go_blog"
download_url "https://blog.golang.org/using-go-modules" "golang_modules"
download_url "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Introduction" "mdn_js_intro"
download_url "https://docs.python.org/3/tutorial/introduction.html" "python_docs"

# Science and research
download_url "https://www.nature.com/articles/d41586-025-00789-1" "nature"
download_url "https://www.scientificamerican.com/article/new-ai-breakthrough-2025/" "scientific_american"

# Wikipedia articles
download_url "https://en.wikipedia.org/wiki/Go_(programming_language)" "wikipedia_go"
download_url "https://en.wikipedia.org/wiki/Artificial_intelligence" "wikipedia_ai"

# Blogs with different layouts
download_url "https://medium.com/better-programming/clean-code-principles-in-go-536472d7c249" "medium_blog"
download_url "https://css-tricks.com/modern-css-in-2025/" "css_tricks"

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

# Download test files from ReadabiliPy repository
echo "Downloading test files from ReadabiliPy repository..."
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/addictinginfo.com-1_full_page.html" "master" "addictinginfo.com-1_full_page.html"
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/addictinginfo.com-1_simple_article_from_full_page.json" "master" "addictinginfo.com-1_simple_article_from_full_page.json"
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/conservativehq.com-1_full_page.html" "master" "conservativehq.com-1_full_page.html"
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/conservativehq.com-1_simple_article_from_full_page.json" "master" "conservativehq.com-1_simple_article_from_full_page.json"
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/davidwolfe.com-1_full_page.html" "master" "davidwolfe.com-1_full_page.html"
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/davidwolfe.com-1_simple_article_from_full_page.json" "master" "davidwolfe.com-1_simple_article_from_full_page.json"
download_github_file "alan-turing-institute/ReadabiliPy" "tests/data/benchmarkinghuge.html" "master" "benchmarkinghuge.html"

echo "Download complete. Files saved to the real_world directory and test files downloaded from ReadabiliPy."
