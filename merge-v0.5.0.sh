#!/bin/bash
# Script to merge v0.5.0-fresh branch to main

echo "=== Strigoi v0.5.0 Fresh Start Merge ==="
echo "This will replace main branch with v0.5.0-fresh (orphan branch)"
echo "Archives have been moved to: /home/cy/archives/Strigoi/"
echo ""
echo "WARNING: This will force-push to main branch!"
echo "Press Ctrl+C to cancel, or Enter to continue..."
read

# Fetch latest
git fetch origin

# Option 1: Replace main with v0.5.0-fresh
echo "Replacing main branch with v0.5.0-fresh..."
git checkout v0.5.0-fresh
git branch -D main
git checkout -b main
git push origin main --force

# Create release tag
echo "Creating v0.5.0 release tag..."
git tag -a v0.5.0 -m "Release v0.5.0 - Cobra CLI Framework

Major Features:
- Cobra CLI framework with interactive REPL
- Professional development methodology
- Reduced project size from 1.3GB to 18MB
- Enhanced color-coded interface
- Comprehensive documentation

This release establishes the foundation for Strigoi as a production-ready security validation platform."

git push origin v0.5.0

echo "✅ Done! Strigoi v0.5.0 has been released!"
echo ""
echo "Next steps:"
echo "1. Close issue #1"
echo "2. Create GitHub Release from v0.5.0 tag"
echo "3. Set up GitHub Project board"
echo "4. Create issue templates"