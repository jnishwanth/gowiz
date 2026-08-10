#!/usr/bin/env bash
set -e

COLOR_RESET="\033[0m"
COLOR_BOLD="\033[1m"
COLOR_GREEN="\033[32m"
COLOR_BLUE="\033[34m"
COLOR_RED="\033[31m"
COLOR_YELLOW="\033[33m"

echo -e "${COLOR_BOLD}${COLOR_BLUE}=== gowiz Tiered Verification Pipeline ===${COLOR_RESET}"

# Tier 1: Fast Feedback (Fmt, Vet, Unit Tests)
echo -e "\n${COLOR_BOLD}[Tier 1] Fast Quality & Unit Tests...${COLOR_RESET}"
UNFMT_FILES=$(gofmt -l .)
if [ -n "$UNFMT_FILES" ]; then
    echo -e "${COLOR_RED}Unformatted code detected in:${COLOR_RESET}\n$UNFMT_FILES"
    echo -e "${COLOR_YELLOW}Running gofmt -w . to auto-format...${COLOR_RESET}"
    gofmt -w .
fi

echo -e "Running go vet..."
go vet ./...

echo -e "Running unit tests..."
go test ./...

echo -e "${COLOR_GREEN}✓ Tier 1 Passed!${COLOR_RESET}"

# Tier 2: Race Detector & Integration Safety
echo -e "\n${COLOR_BOLD}[Tier 2] Race Detection & Integration Verification...${COLOR_RESET}"
go test -race ./...

echo -e "${COLOR_GREEN}✓ Tier 2 Passed!${COLOR_RESET}"

# Tier 3: Binary Build & Headless Execution Sanity
echo -e "\n${COLOR_BOLD}[Tier 3] Binary Build & Headless Verification...${COLOR_RESET}"
mkdir -p bin
go build -o bin/gowiz .

./bin/gowiz --check

echo -e "${COLOR_GREEN}✓ Tier 3 Passed!${COLOR_RESET}"

echo -e "\n${COLOR_BOLD}${COLOR_GREEN}🎉 ALL VERIFICATION TIERS PASSED SUCCESSFULLY! Ready for Commit.${COLOR_RESET}\n"
