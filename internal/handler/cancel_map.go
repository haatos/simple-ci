package handler

import (
	"context"
	"sync"
)

func NewCancelMap[K comparable]() *CancelMap[K] {
	return &CancelMap[K]{
		cancels: make(map[K]context.CancelFunc),
	}
}

type CancelMap[K comparable] struct {
	m       sync.Mutex
	cancels map[K]context.CancelFunc
}

func (m *CancelMap[K]) AddCancel(id K, cf context.CancelFunc) {
	m.m.Lock()
	defer m.m.Unlock()
	m.cancels[id] = cf
}

func (m *CancelMap[K]) RemoveCancel(key K) {
	m.m.Lock()
	defer m.m.Unlock()
	delete(m.cancels, key)
}

func (m *CancelMap[K]) Call(key K) {
	m.cancels[key]()
}
