// +build linux

package main

const clientSetupScript = `
#!/bin/sh

set -e
set -u

ifconfig {{.DeviceName}} {{.LocalIP}} {{.RemoteIP}} mtu 1500 up > /dev/null

CURRENT_GATEWAY=$(route -n get default | grep gateway | cut -d ':' -f 2)

route delete default > /dev/null

route add default {{.RemoteIP}} > /dev/null

echo $CURRENT_GATEWAY
`

const clientShutdownScript = `
#!/bin/sh

set -e
set -u

route delete default || true

route add default {{.GatewayIP}}
`

const serverSetupScript = ""

const serverShutdownScript = ""
