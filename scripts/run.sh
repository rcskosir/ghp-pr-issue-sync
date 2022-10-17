#!/bin/sh

echo
echo "Job started: $(date)"
ghp-pr-sync
echo "Job finished: $(date)"
