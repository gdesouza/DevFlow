#!/bin/bash

# Script to find the latest release tag in git history

latest_release=$(git tag --sort=-version:refname | head -1)

if [ -z "$latest_release" ]; then
    echo "No releases found"
else
    echo "$latest_release"
fi