// +build linux

package main

const clientSetupScript = `
#!/bin/sh

echo "Linux client is not supported"
exit 1
`

const clientShutdownScript = `
#!/bin/sh

echo "Linux client is not supported"
exit 1
`

const serverSetupScript = `
#!/bin/sh

`
const serverShutdownScript = `
#!/bin/sh

`
