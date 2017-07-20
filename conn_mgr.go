package main

import (
	"net"
	"sync"
)

// ConnMgr manages client connections by NAT IP
type ConnMgr struct {
	s map[string]net.Conn
	l *sync.RWMutex
}

// NewConnMgr new connection manager
func NewConnMgr() *ConnMgr {
	return &ConnMgr{
		s: make(map[string]net.Conn),
		l: &sync.RWMutex{},
	}
}

// Set set k-v
func (m *ConnMgr) Set(k net.IP, v net.Conn) {
	m.l.Lock()
	defer m.l.Unlock()
	m.s[k.String()] = v
}

// Get get k-v
func (m *ConnMgr) Get(k net.IP) net.Conn {
	m.l.RLock()
	defer m.l.RUnlock()
	return m.s[k.String()]
}

// Delete delete a key
func (m *ConnMgr) Delete(k net.IP) {
	m.l.Lock()
	defer m.l.Unlock()
	delete(m.s, k.String())
}
