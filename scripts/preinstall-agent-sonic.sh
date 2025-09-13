#!/bin/bash
if ! id jackadi >/dev/null 2>&1; then
  useradd --system --home-dir /var/lib/jackadi --shell /bin/false --groups admin jackadi
else
  if ! groups jackadi | grep -q "\badmin\b"; then
    usermod -a -G admin jackadi
  fi
fi