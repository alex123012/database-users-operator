package connection

import (
	"context"
	"sync"
)

type FakeConnection struct {
	queries map[string]int
	list    []string
	count   int
	lock    *sync.RWMutex
}

func NewFakeConnection() *FakeConnection {
	return &FakeConnection{
		lock: &sync.RWMutex{},
	}
}

func (m *FakeConnection) Connect(_ context.Context, _, _ string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.queries != nil {
		return nil
	}

	m.queries = make(map[string]int)
	return nil
}

func (m *FakeConnection) Close(_ context.Context) error {
	return nil
}

func (m *FakeConnection) Copy() interface{} {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m
}

func (m *FakeConnection) Exec(_ context.Context, _ LogInfo, query string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.count++
	m.queries[query] = m.count
	m.list = append(m.list, query)
	return nil
}

func (m *FakeConnection) Queries() map[string]int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.queries
}

func (m *FakeConnection) QueriesList() []string {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.list
}

func (m *FakeConnection) SetDB(db map[string]int) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.queries = db
	m.count = 0
}

func (m *FakeConnection) SetLock(lock *sync.RWMutex) {
	m.lock = lock
}
