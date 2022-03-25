#!/bin/sh

PROXY_PORT=${PROXY_PORT:-8000}
METADATA_IP=${METADATA_IP:-169.254.169.254}
METADATA_PORT=${METADATA_PORT:-80}
PROXY_UID=${PROXY_UID:-1501}

iptables -t nat -N AZWI_PROXY_OUTPUT
iptables -t nat -N AZWI_PROXY_REDIRECT

# Redirect all TCP traffic for metatadata endpoint to the proxy
iptables -t nat -A AZWI_PROXY_REDIRECT -p tcp -j REDIRECT --to-port "${PROXY_PORT}"
# For outbound TCP traffic to metadata endpoint on port 80 jump from OUTPUT chain to AZWI_PROXY_OUTPUT chain
iptables -t nat -A OUTPUT -p tcp -d "${METADATA_IP}" --dport "${METADATA_PORT}" -j AZWI_PROXY_OUTPUT
# Skip redirection of proxy traffic back to itself, return to next chain for further processing
iptables -t nat -A AZWI_PROXY_OUTPUT -m owner --uid-owner "${PROXY_UID}" -j ACCEPT
# For all other traffic to metadata point, jump to AZWI_PROXY_REDIRECT chain
iptables -t nat -A AZWI_PROXY_OUTPUT -j AZWI_PROXY_REDIRECT

# List all iptables rules
iptables -t nat --list
