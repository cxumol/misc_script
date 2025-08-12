#!/bin/sh
# example: curl -sL https://github.com/cxumol/misc_script/raw/refs/heads/master/LLM/catfs.sh | bash -s -- 'src/**/*.py'
set -e
OUTPUT_PATH = catfs.txt
{
  for file in "$@"; do
    if [ -f "$file" ]; then
      echo "$file"
      echo
      echo "\`\`\`${file##*.}"
      cat "$file"
      echo "\`\`\`"
      echo
    fi
  done
} > $OUTPUT_PATH
