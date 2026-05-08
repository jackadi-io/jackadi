#!/bin/bash
systemctl stop jackadi-node.service 2>/dev/null || true
systemctl disable jackadi-node.service 2>/dev/null || true