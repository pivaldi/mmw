# How To Stow

## The Concept of a "Package" in Stow

In the context of Stow, a **Package** is a discrete set of software (or in this case, scripts) stored in a central location.
The critical concept to grasp is that Stow expects the internal structure of a package to **exactly mirror** the structure of the target destination.

### How it works

* **The Stow Directory**: This is your "source of truth" (e.g., `poc/stow-source`).
* **The Package**: A folder inside the Stow Directory (e.g., `common`).
* **The Target**: The service where the scripts should appear (e.g., `poc/services/todo`).

When you "stow" the package, Stow creates **symbolic links** in the target.
If your package contains a folder named `scripts/`, Stow will ensure a folder named `scripts/` appears in your service.

## Recommended Directory Structure

To keep scripts in a `/scripts` folder within each service, organize your root project like this:

```text
poc/ (Root)
├── stow-central/           <-- The "Stow Directory"
│   └── common/             <-- The "Package" name
│       └── scripts/        <-- Subfolder 1 (Target matches this)
│           └── common/     <-- Subfolder 2 (This becomes the link)
│               ├── db-wait.sh
│               └── migrate.sh
└── services/
    └── todo/               <-- The "Target"
        └── scripts/        <-- Stow will place the "common" link here
```

## How to Share the Scripts

Follow these steps to link the central scripts into the `todo` service.

- Central Scripts: `stow-central/common/scripts/common/*.sh`
- Share the Stow storage directory with the `todo` service as the target:
  ```bash
  cd stow-central
  stow --target=../services/todo common
  ```
- Verify the Symlinks
  ```bash
  > ls -l ../services/todo/scripts/
  # common ## symlink if this directory does not pre-exist otherwise it's a symlink (better)
  # todo-start.sh
  #…

  > ls -l ../services/todo/scripts/common
  # db-wait.sh -> ../../../stow-central/common/scripts/common/db-wait.sh
  #…
```

## Automation with `mise`

You can automate the "stowing" process so it happens automatically for all services in your monorepo.

**Add this task to your root `mise.toml`:**

```toml
[tasks."setup:scripts"]
description = "Sync central scripts to all service modules"
run = """
[ -e "$dir/scripts/" ] || mkdir -p "$dir/scripts/"
for dir in services/*/; do
  # -D unlinks first to prevent conflicts, then stow relinks
  stow --dir=stow-central --target="$dir" -D common
  stow --dir=stow-central --target="$dir" common
done
"""
```

```
poc/
├── stow-central/
│   └── common/             <-- The Package
│       ├── .golangci-lint.yaml  <-- PLACE IT HERE
│       └── scripts/
│           └── shared/
└── services/
    └── todo/               <-- The Target
        ├── .golangci-lint.yaml  <-- SYMLINK created here
        └── scripts/
```

## De-Stowing

```bash
#!/bin/bash

# Target directory (defaults to current directory)
TARGET_DIR=${1:-"."}
# The absolute path to your central stow storage
STOW_ABS_PATH=$(readlink -f "stow-central")

echo "Materializing ONLY Stow-managed links in: $TARGET_DIR"

find "$TARGET_DIR" -type l | while read -r link; do
    # Get the real path of the link
    REAL_PATH=$(readlink -f "$link")

    # CHECK: Does the real path start with our stow-central path?
    if [[ "$REAL_PATH" == "$STOW_ABS_PATH"* ]]; then
        echo "Valid Stow link found: $link -> $REAL_PATH"

        # Remove the link
        rm "$link"

        # Replace with physical copy
        if [ -d "$REAL_PATH" ]; then
            cp -r "$REAL_PATH" "$link"
        else
            cp "$REAL_PATH" "$link"
        fi
        echo "Successfully materialized: $link"
    else
        echo "Skipping non-Stow link: $link (points to $REAL_PATH)"
    fi
done
```
