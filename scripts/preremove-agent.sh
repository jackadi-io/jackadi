#!/bin/bash
systemctl stop jackadi-agent.service 2>/dev/null || true
systemctl disable jackadi-agent.service 2>/dev/null || true