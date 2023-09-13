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

package postgresql

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

func escapeLiteral(str string) string {
	ident := strings.Split(str, ".")
	parts := make([]string, len(ident))
	for i := range ident {
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(ident[i], string([]byte{0}), ""), `"`, `""`)

		if parts[i] != "*" {
			parts[i] = `"` + parts[i] + `"`
		}
	}
	return strings.Join(parts, ".")
}

func escapeString(str string) string {
	return "'" + strings.ReplaceAll(str, "'", "''") + "'"
}

func escapeLiteralWithoutQuotes(str string) string {
	ident := strings.Split(str, ".")
	parts := make([]string, len(ident))
	for i := range ident {
		tmp := strings.ReplaceAll(ident[i], string([]byte{0}), "")
		tmp = strings.ReplaceAll(tmp, `"`, `""`)
		parts[i] = strings.ReplaceAll(tmp, `#`, ``)
		parts[i] = strings.ReplaceAll(tmp, `;`, ``)
	}
	return strings.Join(parts, ".")
}

func ignoreAlreadyExists(err error) error {
	if isAlreadyExists(err) {
		return nil
	}
	return err
}

func isAlreadyExists(err error) bool {
	return ProcessToPostgressError(err) == "42710"
}

func ignoreNotExists(err error) error {
	if isNotExists(err) {
		return nil
	}
	return err
}

func isNotExists(err error) bool {
	return ProcessToPostgressError(err) == "42704"
}

const notAPostgresError string = "Not a postgres error"

func ProcessToPostgressError(err error) string {
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return pgErr.SQLState()
		}
		return notAPostgresError
	}
	return ""
}
