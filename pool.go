package main

import (
	"container/list"
	"time"
)

type PlivoPool struct {
	connections *list.List
}

func NewPlivoPool() *PlivoPool {
	return &PlivoPool{list.New()}
}

//TODO before add test conection
func (pool *PlivoPool) Add(URL, SID, AuthToken string, limit uint64) error {
	pool.connections.PushBack(&PlivoConnection{
		URL:       URL,
		SID:       SID,
		AuthToken: AuthToken,
		Limit:     limit,
	})

	return nil
}

func (pool *PlivoPool) Get() <-chan *PlivoConnection {
	connFind := make(chan *PlivoConnection)
	go func() {
		for iter := pool.connections.Front(); iter != nil; iter.Next() {
			conn := iter.Value.(*PlivoConnection)
			if !conn.ReachedLimit() {
				connFind <- conn
			}
			time.Sleep(time.Second)
		}

	}()

	return connFind
}
