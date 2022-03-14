#!/bin/bash
echo "Run this from repo root using ./scripts/dev/golint.sh"
cd ./internal
for d in */ ; do
    echo "---internal/${d}---"
    golint ${d}
done
