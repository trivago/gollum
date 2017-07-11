package patterns

// Redis is a collection of patterns used for Redis logs.
// See https://redis.io
var Redis = map[string]string{
	"REDISTIMESTAMP": `%{MONTHDAY} %{MONTH} %{TIME}`,
	"REDISLOG":       `\[%{POSINT:pid}\] %{REDISTIMESTAMP:timestamp} \* `,
}
