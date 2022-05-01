#!/bin/bash

# Copyright Red H

set -e

echo "Running bundle check..."

STATUS=$( git status --porcelain bundle )

if [ ! -z "$STATUS" ]; then
    echo "FAILED: 'make bundle' ran and modified bundle files in bundle folder. Please remove modified bundle files from git and do not run 'make bundle'.  "
    exit 1
fi
echo "##### bundle check #### Success"
exit 0