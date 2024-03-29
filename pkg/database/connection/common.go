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
)

type LogInfo int

const (
	DisableLogger LogInfo = 1
	EnableLogger  LogInfo = 0
)

type Connection interface {
	Copy() Connection
	Close(ctx context.Context) error
	Connect(ctx context.Context, driver string, connString string) error
	Exec(ctx context.Context, disableLog LogInfo, query string, args ...interface{}) error
}
