#!/bin/bash
systemctl daemon-reload
if [ "$1" = "purge" ] || [ "$1" = "0" ]; then
  userdel jackadi 2>/dev/null || true
  rm -rf /etc/jackadi /opt/jackadi/plugins /var/lib/jackadi /run/jackadi
fi