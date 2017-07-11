package patterns

// PostgreSQL is a collection of patterns to process the pg_log format.
var PostgreSQL = map[string]string{
	"POSTGRESQL": `%{DATESTAMP:timestamp} %{TZ} %{DATA:user_id} %{GREEDYDATA:connection_id} %{POSINT:pid}`,
}
