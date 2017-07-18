// +build !linux,!darwin

package main

import (
	"errors"
)

func (c *Client) setupTUN() (err error) {
	err = errors.New("platform not supported")
	return
}
