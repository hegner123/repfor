#!/usr/bin/env bash

set -e

echo "Building repfor..."
go build -o repfor

echo "Installing repfor to /usr/local/bin..."
sudo cp repfor /usr/local/bin/

echo "Verifying installation..."
if command -v repfor &> /dev/null; then
    echo "repfor installed successfully!"
    echo "Version: $(repfor --cli --search test --replace test 2>&1 | head -1 || echo 'repfor v1.0.0')"
else
    echo "Error: repfor installation failed"
    exit 1
fi

echo ""
echo "To use repfor as an MCP server in Claude Code, add this to your .mcp.json:"
echo ""
echo '{'
echo '  "mcpServers": {'
echo '    "repfor": {'
echo '      "command": "repfor"'
echo '    }'
echo '  }'
echo '}'
