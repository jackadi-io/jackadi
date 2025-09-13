#!/bin/bash
systemctl daemon-reload
systemctl enable jackadi-agent.service
echo "Jackadi Agent installed. Start with: systemctl start jackadi-agent"