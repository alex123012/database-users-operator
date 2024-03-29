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

package database

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alex123012/database-users-operator/api/v1alpha1"
	"github.com/alex123012/database-users-operator/pkg/database/connection"
)

type fakeConnection interface {
	connection.Connection
	Queries() map[string]int
	Connections() map[string]bool
	ResetDB()
}

type FakeDatabase struct {
	Conn fakeConnection
}

func NewFakeDatabase() *FakeDatabase {
	return &FakeDatabase{
		Conn: connection.NewFakeConnection(),
	}
}

func (f *FakeDatabase) DatabaseCreatorFunc() func(context.Context, v1alpha1.DatabaseSpec, client.Client, logr.Logger) (Database, error) {
	return func(ctx context.Context, s v1alpha1.DatabaseSpec, client client.Client, logger logr.Logger) (Database, error) {
		return newDatabase(ctx, f.Conn, s, client, logger)
	}
}
