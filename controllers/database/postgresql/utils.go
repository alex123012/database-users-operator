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
