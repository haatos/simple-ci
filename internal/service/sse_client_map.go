package service

import (
	"sync"
)

func NewSSEClientMap[T any]() *SSEClientMap[T] {
	return &SSEClientMap[T]{
		clients: make(map[string]chan T),
	}
}

type SSEClientMap[T any] struct {
	m       sync.Mutex
	clients map[string]chan T
}

func (cm *SSEClientMap[T]) AddClient(uid string) {
	cm.m.Lock()
	defer cm.m.Unlock()
	cm.clients[uid] = make(chan T)
}

func (cm *SSEClientMap[T]) RemoveClient(uid string) {
	cm.m.Lock()
	defer cm.m.Unlock()
	close(cm.clients[uid])
	delete(cm.clients, uid)
	if len(cm.clients) == 0 {
		cm.clients = make(map[string]chan T)
	}
}

func (cm *SSEClientMap[T]) SendToClients(message T) {
	cm.m.Lock()
	defer cm.m.Unlock()
	for i := range cm.clients {
		cm.clients[i] <- message
	}
}

func (cm *SSEClientMap[T]) GetClient(id int64, uid string) chan T {
	return cm.clients[uid]
}
