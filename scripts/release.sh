#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
error() { echo -e "${RED}Error: $1${NC}" >&2; exit 1; }
info() { echo -e "${BLUE}$1${NC}"; }
success() { echo -e "${GREEN}$1${NC}"; }
warn() { echo -e "${YELLOW}$1${NC}"; }

# Check prerequisites
check_prerequisites() {
    info "Checking prerequisites..."
    
    # Check gh CLI
    if ! command -v gh &> /dev/null; then
        error "gh CLI is not installed. Install from https://cli.github.com/"
    fi
    
    # Check gh authentication
    if ! gh auth status &> /dev/null; then
        error "gh CLI is not authenticated. Run 'gh auth login' first."
    fi
    
    # Check we're in a git repo
    if ! git rev-parse --git-dir &> /dev/null; then
        error "Not in a git repository"
    fi
    
    # Check for uncommitted changes (ignoring .beads/ local state)
    if git diff-index HEAD -- | grep -v "^.beads/" | grep -q .; then
        error "You have uncommitted changes. Please commit or stash them first."
    fi
    
    # Check we're on main branch
    CURRENT_BRANCH=$(git branch --show-current)
    if [[ "$CURRENT_BRANCH" != "main" && "$CURRENT_BRANCH" != "master" ]]; then
        warn "Warning: You're on branch '$CURRENT_BRANCH', not main/master"
        read -p "Continue anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    # Check if local is up to date with remote
    git fetch origin --quiet
    LOCAL=$(git rev-parse HEAD)
    REMOTE=$(git rev-parse @{u} 2>/dev/null || echo "")
    if [[ -n "$REMOTE" && "$LOCAL" != "$REMOTE" ]]; then
        error "Local branch is not up to date with remote. Run 'git pull' first."
    fi
    
    success "All prerequisites met!"
}

# Get last version
get_last_version() {
    git tag -l 'v*' --sort=-v:refname | head -1
}

# Validate semver format
validate_version() {
    local version=$1
    if [[ ! $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
        return 1
    fi
    return 0
}

# Suggest next version based on last version
suggest_next_version() {
    local last=$1
    if [[ -z "$last" ]]; then
        echo "v0.1.0"
        return
    fi
    
    # Extract major.minor.patch
    local version=${last#v}
    local major minor patch
    IFS='.' read -r major minor patch <<< "${version%%-*}"
    
    # Increment patch by default
    patch=$((patch + 1))
    echo "v${major}.${minor}.${patch}"
}

# Main release flow
main() {
    echo ""
    echo "=========================================="
    echo "         Gendocs Release Script          "
    echo "=========================================="
    echo ""
    
    check_prerequisites
    echo ""
    
    # Get and display last version
    LAST_VERSION=$(get_last_version)
    if [[ -n "$LAST_VERSION" ]]; then
        info "Last version: $LAST_VERSION"
    else
        warn "No previous version found. This will be the first release."
    fi
    
    # Suggest next version
    SUGGESTED=$(suggest_next_version "$LAST_VERSION")
    echo ""
    
    # Ask for new version
    read -p "Enter new version [$SUGGESTED]: " NEW_VERSION
    NEW_VERSION=${NEW_VERSION:-$SUGGESTED}
    
    # Ensure version starts with 'v'
    if [[ ! $NEW_VERSION =~ ^v ]]; then
        NEW_VERSION="v$NEW_VERSION"
    fi
    
    # Validate version format
    if ! validate_version "$NEW_VERSION"; then
        error "Invalid version format. Use semantic versioning: vX.Y.Z (e.g., v1.0.0, v2.1.3-beta)"
    fi
    
    # Check if tag already exists
    if git tag -l "$NEW_VERSION" | grep -q .; then
        error "Tag $NEW_VERSION already exists!"
    fi
    
    echo ""
    info "Release Summary:"
    echo "  Previous version: ${LAST_VERSION:-none}"
    echo "  New version:      $NEW_VERSION"
    echo ""
    
    # Show commits since last version
    if [[ -n "$LAST_VERSION" ]]; then
        info "Changes since $LAST_VERSION:"
        git log --oneline "$LAST_VERSION"..HEAD | head -20
        COMMIT_COUNT=$(git rev-list --count "$LAST_VERSION"..HEAD)
        if [[ $COMMIT_COUNT -gt 20 ]]; then
            echo "  ... and $((COMMIT_COUNT - 20)) more commits"
        fi
        echo ""
    fi
    
    # Confirm
    read -p "Create release $NEW_VERSION? (y/N) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        warn "Release cancelled."
        exit 0
    fi
    
    echo ""
    info "Creating release..."
    
    # Create and push tag
    info "Creating tag $NEW_VERSION..."
    git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION"
    
    info "Pushing tag to origin..."
    git push origin "$NEW_VERSION"
    
    # Generate release notes
    RELEASE_NOTES=""
    if [[ -n "$LAST_VERSION" ]]; then
        RELEASE_NOTES=$(git log --pretty=format:"- %s" "$LAST_VERSION"..HEAD)
    else
        RELEASE_NOTES=$(git log --pretty=format:"- %s" HEAD~10..HEAD 2>/dev/null || git log --pretty=format:"- %s")
    fi
    
    # Create GitHub release
    info "Creating GitHub release..."
    gh release create "$NEW_VERSION" \
        --title "$NEW_VERSION" \
        --notes "$RELEASE_NOTES" \
        --generate-notes
    
    echo ""
    success "Release $NEW_VERSION created successfully!"
    echo ""
    info "GitHub Actions will now build and attach binaries."
    info "View release: $(gh release view "$NEW_VERSION" --json url -q .url)"
    echo ""
}

main "$@"
