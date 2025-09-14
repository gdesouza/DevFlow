#!/bin/bash

# Script to bump version based on latest git release tag
# Usage: ./bump-version.sh <major|minor|patch>

bump_type=$1

if [[ "$bump_type" != "major" && "$bump_type" != "minor" && "$bump_type" != "patch" ]]; then
    echo "Error: Invalid bump type. Use major, minor, or patch."
    exit 1
fi

# Get the latest release tag
latest_release=$(git tag --sort=-version:refname | head -1)

if [ -z "$latest_release" ]; then
    # No versions found, start with 0.0.0 and bump
    current_version="0.0.0"
else
    # Assume tag starts with v, like v1.2.3
    current_version=${latest_release#v}
fi

# Parse version
IFS='.' read -r major minor patch <<< "$current_version"

# Bump based on type
case $bump_type in
    major)
        major=$((major + 1))
        minor=0
        patch=0
        ;;
    minor)
        minor=$((minor + 1))
        patch=0
        ;;
    patch)
        patch=$((patch + 1))
        ;;
esac

# Output new version
echo "$major.$minor.$patch"