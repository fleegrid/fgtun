// +build linux

package main

const clientSetupScript = `
#!/bin/sh

set -e
set -u

ifconfig {{.DeviceName}} {{.LocalIP}} pointopoint {{.RemoteIP}} mtu {{.MTU}} netmask {{.Netmask}} up

CURRENT_GATEWAY=$(/sbin/ip route | awk '/default/ { print $3 }')

route delete default

route add default gw {{.RemoteIP}}

echo "------------"
echo $CURRENT_GATEWAY
`

const clientShutdownScript = `
#!/bin/sh

route delete default || true

route add default gw {{.GatewayIP}}
`

const serverSetupScript = `
#!/bin/sh

set -e
set -u

ifconfig {{.DeviceName}} {{.LocalIP}} pointopoint {{.RemoteIP}} mtu {{.MTU}} netmask {{.Netmask}} up

route add -net {{.CIDR}} gw {{.RemoteIP}}

iptables -t nat -A POSTROUTING -o {{.DeviceName}} -j MASQUERADE

echo 1 > /proc/sys/net/ipv4/ip_forward
`

const serverShutdownScript = `
#!/bin/sh

ifconfig {{.DeviceName}} down

route delete {{.CIDR}}
`
