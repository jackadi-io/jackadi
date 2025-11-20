#!/bin/bash
systemctl daemon-reload
if [ "$1" = "purge" ] || [ "$1" = "0" ]; then
  rm -rf /var/lib/jackadi /etc/jackadi
fi