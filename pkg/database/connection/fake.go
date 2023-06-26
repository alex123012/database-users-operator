/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package connection

import (
	"context"
	"fmt"
	"sync"
)

type FakeConnection struct {
	queries     map[string]int
	count       int
	connections map[string]bool
	lock        *sync.RWMutex
}

func NewFakeConnection() *FakeConnection {
	return &FakeConnection{
		lock: &sync.RWMutex{},
	}
}

func (m *FakeConnection) Connect(_ context.Context, driver, connString string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.queries == nil {
		m.queries = make(map[string]int)
	}

	if m.connections == nil {
		m.connections = make(map[string]bool)
	}

	m.connections[fmt.Sprintf("%s:%s", driver, connString)] = true

	return nil
}

func (m *FakeConnection) Close(_ context.Context) error {
	return nil
}

func (m *FakeConnection) Copy() Connection {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m
}

func (m *FakeConnection) Exec(_ context.Context, _ LogInfo, query string, args ...interface{}) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	q := fmt.Sprint(append([]interface{}{query}, args...)...)
	m.count++
	m.queries[q] = m.count
	return nil
}

func (m *FakeConnection) Queries() map[string]int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.queries
}

func (m *FakeConnection) Connections() map[string]bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.connections
}

func (m *FakeConnection) ResetDB() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.queries = make(map[string]int)
	m.connections = make(map[string]bool)
}
