#!/bin/bash

# This script searches for plugin .go sources and outputs a list of corresponding .rst file names

modules_dir="$1"
target_dir="$2"
exclude="$3"

set -x
find "${modules_dir}" -maxdepth 1 -type f -name '*.go' -not -name '*_test.go' |
    grep -vE "${exclude}" |
    sed "s^${modules_dir}^${target_dir}^"  |
    sed 's|\.go$|.rst|' |
    sort --ignore-case
    
