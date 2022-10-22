package database

type DBDriver string

const (
	DBDriverPostgres DBDriver = "pgx"
)

type any = interface{}

type NamedQuery struct {
	Query string
	Arg   any
}

type Query struct {
	Query string
	Args  []any
}

type PostgresSSLMode string

const (
	SSLModeDISABLE    PostgresSSLMode = "disable"
	SSLModeALLOW      PostgresSSLMode = "allow"
	SSLModePREFER     PostgresSSLMode = "prefer"
	SSLModeREQUIRE    PostgresSSLMode = "require"
	SSLModeVERIFYCA   PostgresSSLMode = "verify-ca"
	SSLModeVERIFYFULL PostgresSSLMode = "verify-full"
)
