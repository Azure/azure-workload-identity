#!/bin/sh

# Forward outbound traffic for 169.254.169.254:80 to proxy
iptables -t nat -A OUTPUT -p tcp -d 169.254.169.254 --dport 80 -j REDIRECT --to-port 8000

# List all iptables rules.
iptables -t nat --list
