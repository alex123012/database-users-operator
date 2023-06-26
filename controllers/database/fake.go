package database

import (
	"context"
	"sync"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/controllers/database/connection"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type fakeConnection interface {
	dbConnection
	Queries() map[string]int
	QueriesList() []string
}

type FakeDatabase struct {
	Conn fakeConnection
	DB   map[string]int
	Lock *sync.RWMutex
}

func NewFakeDatabase() *FakeDatabase {
	conn := connection.NewFakeConnection()

	db := make(map[string]int)
	conn.SetDB(db)

	lock := &sync.RWMutex{}
	conn.SetLock(lock)

	return &FakeDatabase{
		DB:   db,
		Lock: lock,
		Conn: conn,
	}
}

func (f *FakeDatabase) DatabaseCreatorFunc() func(context.Context, v1alpha1.DatabaseSpec, client.Client, logr.Logger) (Interface, error) {
	return func(ctx context.Context, s v1alpha1.DatabaseSpec, kClient client.Client, logger logr.Logger) (Interface, error) {
		db, err := newDatabase(ctx, f.Conn, s, kClient, logger)
		return db, err
	}
}

func (f *FakeDatabase) ResetDB() {
	f.Lock.Lock()
	defer f.Lock.Unlock()
	f.DB = make(map[string]int)
}
