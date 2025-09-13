#!/bin/bash
systemctl stop jackadi-manager.service 2>/dev/null || true
systemctl disable jackadi-manager.service 2>/dev/null || true