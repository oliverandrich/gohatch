#!/usr/bin/env bash
# Generates llms-full.txt by concatenating all documentation markdown files
# in a logical order. Output goes to docs/llms-full.txt.
set -euo pipefail

DOCS_DIR="docs"
OUTPUT="$DOCS_DIR/llms-full.txt"

# Files in reading order.
files=(
    index.md
    usage.md
    templates.md
    creating-templates.md
)

{
    echo "# gohatch — Full Documentation"
    echo ""
    echo "> Complete documentation for gohatch, a project scaffolding tool for Go."
    echo ""

    for file in "${files[@]}"; do
        path="$DOCS_DIR/$file"
        if [[ -f "$path" ]]; then
            echo "---"
            echo ""
            cat "$path"
            echo ""
        fi
    done
} > "$OUTPUT"

wc -l < "$OUTPUT" | xargs printf "Generated %s (%s lines)\n" "$OUTPUT"
