#!/bin/bash
systemctl daemon-reload
if [ "$1" = "purge" ] || [ "$1" = "0" ]; then
  userdel jackadi 2>/dev/null || true
  rm -rf /var/lib/jackadi /etc/jackadi
fi