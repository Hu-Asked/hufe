#!/usr/bin/env bash
export HUFE_OUTPUT=$(mktemp)

/home/hueth/documents/hufe/hufe

clear

if [ -s "$HUFE_OUTPUT" ]; then
    TARGET_DIR=$(cat "$HUFE_OUTPUT")
    rm "$HUFE_OUTPUT"
fi

# if [[ -z "$TARGET_DIR" ]]; then
#     echo "Error: No path returned"
#     return 1
# fi

if [[ -d "$TARGET_DIR" ]]; then
    cd "$TARGET_DIR"
    unset TARGET_DIR
else
    # echo "$TARGET_DIR"
    # echo "Error: Invalid path '$TARGET_DIR'"
    unset TARGET_DIR
    return 1
fi

