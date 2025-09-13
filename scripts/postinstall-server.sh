#!/bin/bash
systemctl daemon-reload
systemctl enable jackadi-manager.service
echo "Jackadi Manager installed. Start with: systemctl start jackadi-manager"