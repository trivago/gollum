#!/bin/bash

gogroup=$1
rstgroup=$2

grep -E "^${gogroup}\." docs_assignment_sorted_person.txt  |
    grep -E '[[:space:]]DONE$' |
    awk '{print $1}' |
    sed -E 's/.*\.//'  | while read name ; do
    gofile=../${gogroup}/${name}.go
    if [ -e $gofile ]; then
        echo src/gen/${rstgroup}/${name}.rst
    else
        echo ERROR: $gofile not found >&2 ;
    fi ;
done
