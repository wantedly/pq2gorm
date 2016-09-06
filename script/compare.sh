#!/bin/bash

for f in `ls testdata/models`; do
  diff -u out/$f testdata/models/$f

  if [[ $? -gt 0 ]]; then
    echo ""
    echo "FAILED: $f does not match."
    echo ""
    exit 1
  fi
done

echo ""
echo "SUCCESS!"
echo ""
