#!/bin/bash
if ! id jackadi >/dev/null 2>&1; then
  useradd --system --home-dir /var/lib/jackadi --shell /bin/false jackadi
fi