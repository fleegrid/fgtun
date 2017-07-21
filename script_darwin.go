// +build darwin

package main

const clientSetupScript = `
#!/bin/sh

set -e
set -u

ifconfig {{.DeviceName}} {{.LocalIP}} {{.RemoteIP}} mtu {{.MTU}} netmask {{.Netmask}} up

CURRENT_GATEWAY=$(route -n get default | grep gateway | cut -d ':' -f 2)

route delete default

route add default {{.RemoteIP}}

echo "------------"
echo $CURRENT_GATEWAY
`

const clientShutdownScript = `
#!/bin/sh

set -e
set -u

route delete default || true

route add default {{.GatewayIP}}
`

const serverSetupScript = `
#!/bin/sh

echo "Drawin server is not supported"
exit 1
`

const serverShutdownScript = `
#!/bin/sh

echo "Drawin server is not supported"
exit 1
`
