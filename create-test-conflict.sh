#!/bin/bash
# Script to create a test merge conflict for gxt conflict resolution feature

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== GXT Conflict Resolution Test Setup ===${NC}\n"

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo -e "${RED}Error: Not in a git repository${NC}"
    exit 1
fi

# Check for uncommitted changes
if ! git diff-index --quiet HEAD -- 2>/dev/null; then
    echo -e "${YELLOW}Warning: You have uncommitted changes.${NC}"
    echo -e "${YELLOW}This script will create new files and branches.${NC}"
    read -p "Continue? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Get current branch
ORIGINAL_BRANCH=$(git rev-parse --abbrev-ref HEAD)
echo -e "${BLUE}Current branch: ${GREEN}${ORIGINAL_BRANCH}${NC}\n"

# Create test file with initial content
TEST_FILE="conflict-test.txt"
echo -e "${BLUE}Creating test file: ${TEST_FILE}${NC}"

cat > "$TEST_FILE" << 'EOF'
# Conflict Test File
# This file will be used to create a merge conflict

## Section 1: Introduction
This is the original content that exists in both branches.
It should not cause any conflicts.

## Section 2: Features
- Feature A: Basic functionality
- Feature B: Advanced options
- Feature C: Integration support

## Section 3: Configuration
The configuration will be modified in both branches.
This will create a conflict.

## Section 4: Footer
End of file.
EOF

git add "$TEST_FILE"
git commit -m "Add conflict test file" || echo -e "${YELLOW}File already committed${NC}"

echo -e "\n${BLUE}Creating feature branch: ${GREEN}test-feature${NC}"
git checkout -b test-feature 2>/dev/null || git checkout test-feature

# Modify the file on feature branch
cat > "$TEST_FILE" << 'EOF'
# Conflict Test File
# This file will be used to create a merge conflict

## Section 1: Introduction
This is the original content that exists in both branches.
It should not cause any conflicts.

## Section 2: Features (Feature Branch)
- Feature A: Enhanced with new capabilities
- Feature B: Advanced options with AI
- Feature C: Integration support for multiple platforms
- Feature D: NEW - Real-time synchronization
- Feature E: NEW - Cloud backup

## Section 3: Configuration (Feature Branch)
The configuration has been updated with new settings:
- Database: PostgreSQL
- Cache: Redis
- Queue: RabbitMQ
- API Version: v2.0

## Section 4: Footer
End of file - Updated by feature branch.
EOF

git add "$TEST_FILE"
git commit -m "Feature: Enhance features and configuration"

# Create another test file for multi-file conflicts
MULTI_FILE="multi-conflict.py"
echo -e "\n${BLUE}Creating additional test file: ${MULTI_FILE}${NC}"

cat > "$MULTI_FILE" << 'EOF'
#!/usr/bin/env python3
"""
Multi-file conflict test
"""

class ConfigManager:
    def __init__(self):
        self.version = "2.0"
        self.features = ["feature-x", "feature-y"]

    def get_config(self):
        return {
            "version": self.version,
            "features": self.features,
            "mode": "feature-mode"
        }

if __name__ == "__main__":
    manager = ConfigManager()
    print(manager.get_config())
EOF

git add "$MULTI_FILE"
git commit -m "Feature: Add configuration manager"

# Switch back to original branch
echo -e "\n${BLUE}Switching back to: ${GREEN}${ORIGINAL_BRANCH}${NC}"
git checkout "$ORIGINAL_BRANCH"

# Make conflicting changes on original branch
cat > "$TEST_FILE" << 'EOF'
# Conflict Test File
# This file will be used to create a merge conflict

## Section 1: Introduction
This is the original content that exists in both branches.
It should not cause any conflicts.

## Section 2: Features (Main Branch)
- Feature A: Completely redesigned
- Feature B: Advanced options with ML
- Feature C: Integration support for legacy systems
- Feature X: NEW - Security enhancements
- Feature Y: NEW - Performance optimizations

## Section 3: Configuration (Main Branch)
The configuration has been updated differently:
- Database: MySQL
- Cache: Memcached
- Queue: Kafka
- API Version: v3.0

## Section 4: Footer
End of file - Updated by main branch.
EOF

git add "$TEST_FILE"
git commit -m "Main: Update features and configuration differently"

# Make conflicting changes to the second file
cat > "$MULTI_FILE" << 'EOF'
#!/usr/bin/env python3
"""
Multi-file conflict test
"""

class ConfigManager:
    def __init__(self):
        self.version = "3.0"
        self.features = ["feature-a", "feature-b", "feature-c"]

    def get_config(self):
        return {
            "version": self.version,
            "features": self.features,
            "mode": "production-mode",
            "debug": False
        }

if __name__ == "__main__":
    manager = ConfigManager()
    config = manager.get_config()
    print(f"Configuration: {config}")
EOF

git add "$MULTI_FILE"
git commit -m "Main: Update configuration manager for production"

echo -e "\n${GREEN}=== Setup Complete! ===${NC}\n"
echo -e "${BLUE}Test scenario created:${NC}"
echo -e "  ${GREEN}✓${NC} Created branch: ${GREEN}test-feature${NC}"
echo -e "  ${GREEN}✓${NC} Created conflicting commits on both branches"
echo -e "  ${GREEN}✓${NC} Created ${YELLOW}2 files${NC} with conflicts: ${YELLOW}${TEST_FILE}${NC}, ${YELLOW}${MULTI_FILE}${NC}"
echo -e "  ${GREEN}✓${NC} Currently on: ${GREEN}${ORIGINAL_BRANCH}${NC}\n"

echo -e "${BLUE}To test merge conflict resolution:${NC}"
echo -e "  1. Run: ${GREEN}./gxt${NC} (or ${GREEN}gxt${NC} if installed)"
echo -e "  2. Press ${YELLOW}'b'${NC} to open Branches view (or ${YELLOW}Ctrl+P${NC} and search 'branches')"
echo -e "  3. Navigate to ${GREEN}test-feature${NC} branch"
echo -e "  4. Press ${YELLOW}'m'${NC} to merge"
echo -e "  5. Conflict resolution view will appear!"
echo -e ""
echo -e "${BLUE}To test rebase conflict resolution:${NC}"
echo -e "  1. First checkout test-feature: ${GREEN}git checkout test-feature${NC}"
echo -e "  2. Run: ${GREEN}./gxt${NC}"
echo -e "  3. Press ${YELLOW}'b'${NC} for Branches view"
echo -e "  4. Navigate to ${GREEN}${ORIGINAL_BRANCH}${NC}"
echo -e "  5. Press ${YELLOW}'b'${NC} to rebase"
echo -e ""
echo -e "${BLUE}Expected conflicts in:${NC}"
echo -e "  • ${YELLOW}Section 2${NC} - Different feature lists"
echo -e "  • ${YELLOW}Section 3${NC} - Different configurations"
echo -e "  • ${YELLOW}Python class${NC} - Different versions and settings"
echo -e ""
echo -e "${BLUE}Conflict resolution keys:${NC}"
echo -e "  ${YELLOW}o${NC} - Accept Ours (current branch)"
echo -e "  ${YELLOW}t${NC} - Accept Theirs (incoming branch)"
echo -e "  ${YELLOW}b${NC} - Accept Base (common ancestor)"
echo -e "  ${YELLOW}B${NC} - Accept Both (ours + theirs)"
echo -e "  ${YELLOW}e${NC} - Edit manually in \$EDITOR"
echo -e "  ${YELLOW}s${NC} - Save & next file"
echo -e "  ${YELLOW}C${NC} - Continue merge/rebase"
echo -e "  ${YELLOW}x${NC} - Abort merge/rebase"
echo -e ""
echo -e "${BLUE}To clean up after testing:${NC}"
echo -e "  ${GREEN}./cleanup-test-conflict.sh${NC}"
echo -e "  (or manually: ${GREEN}git checkout ${ORIGINAL_BRANCH} && git branch -D test-feature && rm ${TEST_FILE} ${MULTI_FILE}${NC})"
echo -e ""
