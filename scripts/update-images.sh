#!/usr/bin/env bash
#
# update-images.sh - Automate container image updates based on Flux ImagePolicies
#
# This script:
# - Queries all Flux ImagePolicy resources
# - Compares latest image tags with deployed versions
# - Updates deployment manifests when newer images are available
# - Creates git commits for changes
#

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
CLUSTER_DIR="${CLUSTER_DIR:-./cluster}"
DRY_RUN="${DRY_RUN:-false}"
AUTO_PUSH="${AUTO_PUSH:-false}"

log_info() {
    echo -e "${BLUE}INFO${NC}: $*"
}

log_success() {
    echo -e "${GREEN}SUCCESS${NC}: $*"
}

log_warning() {
    echo -e "${YELLOW}WARNING${NC}: $*"
}

log_error() {
    echo -e "${RED}ERROR${NC}: $*"
}

# Check required commands
for cmd in kubectl jq git; do
    if ! command -v "$cmd" &> /dev/null; then
        log_error "Required command '$cmd' not found"
        exit 1
    fi
done

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    log_error "Not in a git repository"
    exit 1
fi

# Get all ImagePolicy resources
log_info "Fetching ImagePolicy resources..."
POLICIES=$(kubectl get imagepolicy -A -o json)

if [[ $(echo "$POLICIES" | jq '.items | length') -eq 0 ]]; then
    log_warning "No ImagePolicy resources found"
    exit 0
fi

UPDATES_MADE=0
UPDATED_FILES=()

# Process each ImagePolicy
while read -r policy; do
    NAMESPACE=$(echo "$policy" | jq -r '.metadata.namespace')
    NAME=$(echo "$policy" | jq -r '.metadata.name')

    # Check if policy is ready
    READY=$(echo "$policy" | jq -r '.status.conditions[] | select(.type=="Ready") | .status')
    if [[ "$READY" != "True" ]]; then
        log_warning "ImagePolicy $NAMESPACE/$NAME is not ready, skipping"
        continue
    fi

    # Get latest image reference
    LATEST_IMAGE=$(echo "$policy" | jq -r '.status.latestRef.name // empty')
    LATEST_TAG=$(echo "$policy" | jq -r '.status.latestRef.tag // empty')

    if [[ -z "$LATEST_IMAGE" || -z "$LATEST_TAG" ]]; then
        log_warning "ImagePolicy $NAMESPACE/$NAME has no latest image, skipping"
        continue
    fi

    FULL_IMAGE="$LATEST_IMAGE:$LATEST_TAG"
    # Build marker search pattern for grep
    MARKER_PATTERN="imagepolicy.*$NAMESPACE:$NAME"

    log_info "Processing $NAMESPACE/$NAME -> $FULL_IMAGE"

    # Find files with this imagepolicy marker
    FILES=$(grep -rl "$MARKER_PATTERN" "$CLUSTER_DIR" 2>/dev/null || true)

    if [[ -z "$FILES" ]]; then
        log_warning "No files found with marker for $NAMESPACE/$NAME"
        continue
    fi

    # Update each file
    while IFS= read -r file; do
        # Extract current image from the file (match line with image and the marker)
        CURRENT_IMAGE=$(grep -E "image:.*# .*$MARKER_PATTERN" "$file" | sed -E 's/.*image: *"?([^"# ]+)"?.*/\1/' || true)

        if [[ -z "$CURRENT_IMAGE" ]]; then
            log_warning "Could not extract current image from $file"
            continue
        fi

        if [[ "$CURRENT_IMAGE" == "$FULL_IMAGE" ]]; then
            log_info "Image in $file is already up to date ($FULL_IMAGE)"
            continue
        fi

        log_info "Updating $file"
        log_info "  FROM: $CURRENT_IMAGE"
        log_info "  TO:   $FULL_IMAGE"

        if [[ "$DRY_RUN" == "true" ]]; then
            log_warning "[DRY RUN] Would update $file"
            UPDATES_MADE=$((UPDATES_MADE + 1))
            continue
        fi

        # Perform the update using sed
        # The pattern matches: image: <anything> # {"$imagepolicy": "namespace:name"}
        # And replaces the image while preserving the marker
        sed -i.bak -E 's|image: *"?[^"# ]+"?( *# *\{"'"'"'\$imagepolicy": *"'"$NAMESPACE:$NAME"'"\})|image: '"$FULL_IMAGE"'\1|g' "$file"
        rm "${file}.bak"

        UPDATES_MADE=$((UPDATES_MADE + 1))
        UPDATED_FILES+=("$file")
        log_success "Updated $file"

    done <<< "$FILES"
done < <(echo "$POLICIES" | jq -c '.items[]')

# Summary
echo ""
if [[ $UPDATES_MADE -eq 0 ]]; then
    log_success "All images are up to date!"
    exit 0
fi

log_success "Updated $UPDATES_MADE file(s)"

if [[ "$DRY_RUN" == "true" ]]; then
    log_warning "DRY RUN mode - no changes committed"
    exit 0
fi

# Create git commit
log_info "Creating git commit..."

# Add all updated files
for file in "${UPDATED_FILES[@]}"; do
    git add "$file"
done

# Check if there are actually changes to commit
if git diff --cached --quiet; then
    log_warning "No changes to commit (files may have been identical after update)"
    exit 0
fi

# Create commit message
COMMIT_MSG="Update container images

Automated update by update-images.sh script.

Updated images:
$(git diff --cached --name-only | while read -r f; do
    echo "  - $f"
done)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"

git commit -m "$COMMIT_MSG"
log_success "Created git commit"

# Push if auto-push is enabled
if [[ "$AUTO_PUSH" == "true" ]]; then
    log_info "Pushing to remote..."
    git push
    log_success "Pushed to remote"
else
    log_info "Changes committed locally. Run 'git push' to push to remote."
fi
