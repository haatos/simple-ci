package handler

import "sync"

func NewSSEClientMap[T any]() *SSEClientMap[T] {
	return &SSEClientMap[T]{
		clients: make(map[int64]map[string]chan T),
	}
}

type SSEClientMap[T any] struct {
	m       sync.Mutex
	clients map[int64]map[string]chan T
}

func (cm *SSEClientMap[T]) AddMap(id int64) {
	cm.m.Lock()
	defer cm.m.Unlock()
	cm.clients[id] = make(map[string]chan T)
}

func (cm *SSEClientMap[T]) RemoveMap(id int64) {
	cm.m.Lock()
	defer cm.m.Unlock()
	delete(cm.clients, id)
}

func (cm *SSEClientMap[T]) HasMap(id int64) bool {
	_, ok := cm.clients[id]
	return ok
}

func (cm *SSEClientMap[T]) AddClient(id int64, uid string) {
	cm.m.Lock()
	defer cm.m.Unlock()
	cm.clients[id][uid] = make(chan T)
}

func (cm *SSEClientMap[T]) RemoveClient(id int64, uid string) {
	cm.m.Lock()
	defer cm.m.Unlock()
	close(cm.clients[id][uid])
	delete(cm.clients[id], uid)
}

func (cm *SSEClientMap[T]) SendToClients(id int64, message T) {
	cm.m.Lock()
	defer cm.m.Unlock()
	for i := range cm.clients[id] {
		cm.clients[id][i] <- message
	}
}

func (cm *SSEClientMap[T]) GetClient(id int64, uid string) chan T {
	return cm.clients[id][uid]
}
