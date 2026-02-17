#!/bin/bash
# Script to clean up test conflict scenario

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== Cleaning up test conflict scenario ===${NC}\n"

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo -e "${RED}Error: Not in a git repository${NC}"
    exit 1
fi

# Get current branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

# Abort any in-progress merge or rebase
if [ -f ".git/MERGE_HEAD" ]; then
    echo -e "${YELLOW}Aborting in-progress merge...${NC}"
    git merge --abort 2>/dev/null || true
fi

if [ -d ".git/rebase-merge" ] || [ -d ".git/rebase-apply" ]; then
    echo -e "${YELLOW}Aborting in-progress rebase...${NC}"
    git rebase --abort 2>/dev/null || true
fi

# Determine the main branch (try main, master, or current)
MAIN_BRANCH="main"
if ! git rev-parse --verify main >/dev/null 2>&1; then
    if git rev-parse --verify master >/dev/null 2>&1; then
        MAIN_BRANCH="master"
    else
        MAIN_BRANCH="$CURRENT_BRANCH"
    fi
fi

# Switch to main branch if not already there
if [ "$CURRENT_BRANCH" != "$MAIN_BRANCH" ]; then
    echo -e "${BLUE}Switching to ${GREEN}${MAIN_BRANCH}${NC}"
    git checkout "$MAIN_BRANCH" 2>/dev/null || {
        echo -e "${YELLOW}Could not switch to ${MAIN_BRANCH}, staying on ${CURRENT_BRANCH}${NC}"
        MAIN_BRANCH="$CURRENT_BRANCH"
    }
fi

# Delete test branch if it exists
if git rev-parse --verify test-feature >/dev/null 2>&1; then
    echo -e "${BLUE}Deleting branch: ${YELLOW}test-feature${NC}"
    git branch -D test-feature 2>/dev/null || {
        echo -e "${YELLOW}Could not delete test-feature branch${NC}"
    }
else
    echo -e "${YELLOW}test-feature branch not found${NC}"
fi

# Remove test files if they exist
TEST_FILES=("conflict-test.txt" "multi-conflict.py")
for file in "${TEST_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${BLUE}Removing test file: ${YELLOW}${file}${NC}"
        git rm -f "$file" 2>/dev/null || rm -f "$file"
    fi
done

# Reset any commits related to test files (optional - commented out for safety)
# Uncomment if you want to remove the test commits:
# echo -e "${YELLOW}Resetting last 2 commits...${NC}"
# git reset --hard HEAD~2

# Check if there are changes to commit
if ! git diff-index --quiet HEAD -- 2>/dev/null; then
    echo -e "${BLUE}Committing cleanup...${NC}"
    git add -A
    git commit -m "Clean up conflict test files" || true
fi

echo -e "\n${GREEN}=== Cleanup Complete! ===${NC}\n"
echo -e "${BLUE}Current state:${NC}"
echo -e "  Branch: ${GREEN}${MAIN_BRANCH}${NC}"
echo -e "  Test branch deleted: ${GREEN}test-feature${NC}"
echo -e "  Test files removed: ${GREEN}conflict-test.txt, multi-conflict.py${NC}"
echo -e ""
echo -e "${YELLOW}Note:${NC} The commits adding test files are still in history."
echo -e "To remove them completely, run: ${GREEN}git reset --hard HEAD~2${NC}"
echo -e ""
