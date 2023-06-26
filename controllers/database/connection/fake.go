package connection

import (
	"context"
	"sync"
)

type fakeConnection struct {
	queries map[string]int
	list    []string
	count   int
	lock    *sync.RWMutex
}

func NewFakeConnection() *fakeConnection {
	return &fakeConnection{
		lock: &sync.RWMutex{},
	}
}

func (m *fakeConnection) Connect(_ context.Context, _, _ string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.queries != nil {
		return nil
	}

	m.queries = make(map[string]int)
	return nil
}

func (m *fakeConnection) Close(_ context.Context) error {
	return nil
}

func (m *fakeConnection) Copy() interface{} {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m
}

func (m *fakeConnection) Exec(_ context.Context, _ LogInfo, query string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.count++
	m.queries[query] = m.count
	m.list = append(m.list, query)
	return nil
}

func (m *fakeConnection) Queries() map[string]int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.queries
}

func (m *fakeConnection) QueriesList() []string {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.list
}

func (m *fakeConnection) SetDB(db map[string]int) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.queries = db
	m.count = 0
}

func (m *fakeConnection) SetLock(lock *sync.RWMutex) {
	m.lock = lock
}
