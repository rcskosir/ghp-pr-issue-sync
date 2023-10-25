#!/bin/sh

echo
echo "Job started: $(date)"
ghp-repo-sync "$SYNC_CMD"
echo "Job finished: $(date)"
