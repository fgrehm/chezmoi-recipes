#!/bin/bash
set -euo pipefail

if dpkg -s ripgrep &>/dev/null; then
  echo "ripgrep: already installed"
  exit 0
fi

sudo apt-get update -qq
sudo apt-get install -y ripgrep
