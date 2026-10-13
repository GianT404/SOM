#!/usr/bin/env bash
set -euo pipefail

current_tag="${1:?Usage: generate-tui-changelog.sh <tui-tag>}"
prefix="tui-v"
output="cmd/som/changelog.txt"

if [[ "$current_tag" != "$prefix"* ]]; then
    echo "Expected a TUI tag, got: $current_tag" >&2
    exit 1
fi

mapfile -t tags < <(git tag --list "${prefix}*" --sort=-version:refname)

previous_tag=""
found_current=false

for tag in "${tags[@]}"; do
    if [[ "$found_current" == true ]]; then
        previous_tag="$tag"
        break
    fi
    if [[ "$tag" == "$current_tag" ]]; then
        found_current=true
    fi
done

if [[ "$found_current" != true ]]; then
    echo "Current TUI tag not found in local Git history: $current_tag" >&2
    exit 1
fi

mkdir -p "$(dirname "$output")"

if [[ -n "$previous_tag" ]]; then
    range_spec="$previous_tag..$current_tag"
else
    range_spec="$current_tag"
fi

commits="$(git log --no-decorate --format='  • %h %s' "$range_spec" -- internal/)"

{
    printf 'SOM TUI changelog — %s\n' "$current_tag"
    if [[ -n "$previous_tag" ]]; then
        printf 'Changes since %s:\n\n' "$previous_tag"
    else
        printf 'Changes under internal/:\n\n'
    fi

    if [[ -n "$commits" ]]; then
        printf '%s\n' "$commits"
    else
        printf '  • No changes under internal/ in this release.\n'
    fi
} > "$output"

echo "Generated $output for $current_tag"
