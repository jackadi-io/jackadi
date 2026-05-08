#!/bin/bash
systemctl daemon-reload
systemctl enable jackadi-node.service
echo "Jackadi Node installed. Start with: systemctl start jackadi-node"