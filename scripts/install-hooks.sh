#!/usr/bin/env bash
set -e

mkdir -p .git/hooks
chmod +x .githooks/pre-commit
cp .githooks/pre-commit .git/hooks/pre-commit

echo "✓ Git pre-commit hook installed successfully!"
