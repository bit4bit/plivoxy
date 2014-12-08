package main

import (
	"net/url"
	"sync"
)

type PlivoRequest struct {
	Conn           *PlivoConnection
	UUID           string
	Params         url.Values
	HangupURLProxy string
}

type ProxyRequest struct {
	*sync.RWMutex
	proxy map[string]*PlivoRequest
}

func (proxy *ProxyRequest) Set(key string, preq *PlivoRequest) {
	proxy.Lock()
	proxy.proxy[key] = preq
	proxy.Unlock()
}

func (proxy *ProxyRequest) Get(key string) *PlivoRequest {
	proxy.RLock()
	defer proxy.RUnlock()
	return proxy.proxy[key]
}

func (proxy *ProxyRequest) Del(key string) {
	proxy.Lock()
	delete(proxy.proxy, key)
	proxy.Unlock()
}

func NewProxyRequest() *ProxyRequest {
	return &ProxyRequest{&sync.RWMutex{}, make(map[string]*PlivoRequest)}
}
