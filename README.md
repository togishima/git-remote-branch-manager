# git-remote-branch-manager

A command-line tool to interactively manage and delete remote Git branches.

## Overview

`git-remote-branch-manager` provides an interactive interface to view and delete remote Git branches. It leverages `fzf` for fuzzy finding and selection, making it easy to manage your remote branches.

## Features

-   **Interactive Selection**: Use `fzf` to select multiple remote branches for deletion.
-   **Preview**: View `git log` for selected branches in a preview window.
-   **Status Indicators**: Clearly see if a branch is `(merged)`, `(unmerged)`, or `(protected)`.
-   **Protected Branches**: Prevents accidental deletion of `main` and `master` branches (and their remote counterparts).
-   **Confirmation**: Displays selected branches and asks for confirmation before deletion.
-   **Multi-language Support**: Supports English and Japanese.

## Installation

1.  **Prerequisites**:
    -   Go (version 1.24.2 or later)
    -   `fzf` (Fuzzy finder). If you don't have `fzf`, install it first:
        -   **macOS (Homebrew)**: `brew install fzf`
        -   **Linux**: Follow instructions [here](https://github.com/junegunn/fzf#installation)

2.  **Build from source**:

    ```bash
    git clone https://github.com/togishima/git-remote-branch-manager.git
    cd git-remote-branch-manager
    go build -o git-remote-branch-manager .
    ```

3.  **Install as a Git subcommand**:

    Move the compiled executable to a directory in your system's PATH (e.g., `/usr/local/bin`). Ensure the executable is named `git-remote-branch-manager`.

    ```bash
    sudo mv git-remote-branch-manager /usr/local/bin/
    ```

    Now you can run the tool as a Git subcommand:

    ```bash
    git remote-branch-manager
    ```

## Usage

Run the tool in your Git repository:

```bash
git remote-branch-manager
```

This will open an `fzf` interface displaying all remote branches. You can:

-   Navigate with arrow keys.
-   Type to fuzzy search.
-   Press `Tab` or `Shift+Tab` to select multiple branches.
-   Press `Enter` to confirm your selection.

Each branch will be displayed with a status indicator and color:

-   **Green (merged)**: The remote branch has been merged into your current `HEAD`.
-   **Red (unmerged)**: The remote branch has not been merged into your current `HEAD`.
-   **Yellow (protected)**: The remote branch is a protected branch (e.g., `main`, `master`) and cannot be deleted.

After selection, the tool will ask for confirmation before proceeding with the deletion.

### Options

-   `-h`, `--help`: Show help message.
-   `-lang string`: Specify the language (e.g., `en`, `ja`). Defaults to system language if supported.

## Deletion Process

When you confirm the deletion, the tool will execute `git push <remote_name> --delete <branch_name>` for each selected branch. Please be careful as this action is irreversible. Protected branches will be skipped automatically.

## Contributing

Feel free to open issues or pull requests.
