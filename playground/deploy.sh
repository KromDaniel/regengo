#!/bin/bash

# Deploy Regengo Playground to GitHub Pages

set -e

echo "🚀 Deploying Regengo Playground to GitHub Pages..."

# Check if gh-pages branch exists
if git show-ref --verify --quiet refs/heads/gh-pages; then
    echo "✓ gh-pages branch exists"
else
    echo "Creating gh-pages branch..."
    git checkout --orphan gh-pages
    git rm -rf .
    git checkout main -- playground/
    mv playground/* .
    rmdir playground
    git add .
    git commit -m "Initial playground deployment"
    git push origin gh-pages
    git checkout main
    echo "✓ Created gh-pages branch"
    exit 0
fi

# Update gh-pages with latest playground
echo "Updating gh-pages branch..."

# Save current branch
CURRENT_BRANCH=$(git branch --show-current)

# Checkout gh-pages
git checkout gh-pages

# Get latest playground from main
git checkout main -- playground/

# Move files to root
cp -r playground/* .
rm -rf playground/

# Commit changes
git add .
git commit -m "Update playground: $(date '+%Y-%m-%d %H:%M:%S')" || echo "No changes to commit"

# Push to gh-pages
git push origin gh-pages

# Go back to original branch
git checkout "$CURRENT_BRANCH"

echo "✅ Playground deployed!"
echo ""
echo "🌐 Visit: https://kromdaniel.github.io/regengo/"
echo ""
echo "Note: It may take a few minutes for GitHub Pages to update."
