#!/bin/sh

echo
echo "Job started: $(date)"
ghp-repo-sync
echo "Job finished: $(date)"
