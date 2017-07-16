package fgtun

import (
	"github.com/fleegrid/core"
)

// Client represents a FGTune client
type Client struct {
	config *core.Config
	cipher core.Cipher
}

// NewClient create a new client
func NewClient(url string) (*Client, error) {
	config, err := core.ParseConfigFromURL(url)
	if err != nil {
		return nil, err
	}
	cipher, err := core.NewCipher(config.Cipher, config.Passwd)
	if err != nil {
		return nil, err
	}
	return &Client{config: config, cipher: cipher}, nil
}
