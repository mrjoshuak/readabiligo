#!/bin/bash

# Fix imports in test files
cd /Users/joshua/ws/active/readabiligo/test

# Replace extractor.New with readabiligo.New
find . -name "*.go" -exec sed -i '' 's/extractor\.New/readabiligo.New/g' {} \;

# Replace extractor.With* with readabiligo.With*
find . -name "*.go" -exec sed -i '' 's/extractor\.With/readabiligo.With/g' {} \;

# Replace types.ContentType with readabiligo.ContentType
find . -name "*.go" -exec sed -i '' 's/types\.ContentType/readabiligo.ContentType/g' {} \;

# Replace types.ContentType* with readabiligo.ContentType*
find . -name "*.go" -exec sed -i '' 's/types\.ContentType/readabiligo.ContentType/g' {} \;

echo "Import replacements complete."