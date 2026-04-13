#!/bin/bash

iptables -t filter -F
iptables -t nat -F
iptables -t mangle -F

iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE

iptables -A FORWARD -i uesimtun0 -o eth0 -j ACCEPT
iptables -A FORWARD -i eth0 -o uesimtun0 -m state --state RELATED,ESTABLISHED -j ACCEPT

echo "UPF iptables rules configured"
