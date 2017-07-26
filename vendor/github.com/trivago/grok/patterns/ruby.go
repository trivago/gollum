package patterns

// Ruby is a collection of ruby log related patterns.
var Ruby = map[string]string{
	"RUBY_LOGLEVEL": `(?:DEBUG|FATAL|ERROR|WARN|INFO)`,
	"RUBY_LOGGER":   `[DFEWI], \[%{TIMESTAMP_ISO8601:timestamp} #%{POSINT:pid}\] *%{RUBY_LOGLEVEL:loglevel} -- +%{DATA:progname}: %{GREEDYDATA:message}`,
}
