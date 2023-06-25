package postgresql

type DBType = string

const (
	PostgreSQL  DBType = "PostgreSQL"
	CockroachDB DBType = "CockroachDB"
)
