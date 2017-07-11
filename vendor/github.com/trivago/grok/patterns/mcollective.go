package patterns

// MCollective is a collection of common Marionette Collective patterns.
// See https://github.com/puppetlabs/marionette-collective
var MCollective = map[string]string{
	"MCOLLECTIVEAUDIT": `%{TIMESTAMP_ISO8601:timestamp}:`,
	"MCOLLECTIVE":      `., \[%{TIMESTAMP_ISO8601:timestamp} #%{POSINT:pid}\]%{SPACE}%{LOGLEVEL:event_level}`,
}
