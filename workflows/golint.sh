#!/bin/bash
echo "Run this from repo root using ./workflows/golint.sh"
cd ./pkg
for d in */ ; do
    echo "---pkg/${d}---"
    golint ${d}
done
