#!/bin/sh

PROXY_PORT=${PROXY_PORT:-8000}
METADATA_IP=${METADATA_IP:-169.254.169.254}
METADATA_PORT=${METADATA_PORT:-80}

# Forward outbound traffic for metadata endpoint to proxy
iptables -t nat -A OUTPUT -p tcp -d "${METADATA_IP}" --dport "${METADATA_PORT}" -j REDIRECT --to-port "${PROXY_PORT}"

# List all iptables rules
iptables -t nat --list
