#!/bin/bash
# Install pre-commit hooks for kv-stash

set -e

echo "Installing pre-commit hooks for kv-stash..."

# Check if pre-commit is installed
if ! command -v pre-commit &> /dev/null; then
    echo "pre-commit not found. Installing..."

    # Try to install with pip
    if command -v pip &> /dev/null; then
        pip install pre-commit
    elif command -v pip3 &> /dev/null; then
        pip3 install pre-commit
    # Try to install with homebrew on macOS
    elif [[ "$OSTYPE" == "darwin"* ]] && command -v brew &> /dev/null; then
        brew install pre-commit
    else
        echo "Error: Could not install pre-commit. Please install it manually:"
        echo "  pip install pre-commit"
        echo "  # or"
        echo "  brew install pre-commit"
        exit 1
    fi
fi

# Install the hooks
echo "Installing pre-commit hooks..."
pre-commit install

# Run hooks on all files to test
echo "Running pre-commit on all files to test installation..."
pre-commit run --all-files || {
    echo "Warning: Some pre-commit hooks failed. This is expected on first run."
    echo "The hooks have been installed and will run on future commits."
}

echo "✅ Pre-commit hooks installed successfully!"
echo ""
echo "The following hooks will now run automatically on git commits:"
echo "  - golangci-lint (Docker-based linting)"
echo "  - go fmt (code formatting)"
echo "  - go vet (static analysis)"
echo "  - go test (run tests)"
echo "  - trailing whitespace check"
echo "  - end-of-file fixer"
echo "  - YAML validation"
echo ""
echo "To skip hooks for a commit, use: git commit --no-verify"
