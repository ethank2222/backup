#!/bin/bash
# GitHub Actions entry point - calls exact same logic as original workflow

# Run setup (EXACT COPY from original workflow)
"$(dirname "$0")/setup.sh"

# Run main backup (EXACT COPY from original workflow)
"$(dirname "$0")/main.sh" 