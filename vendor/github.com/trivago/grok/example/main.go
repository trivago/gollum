package main

import (
	"fmt"

	"github.com/trivago/grok"
)

func main() {
	fmt.Println("# Default Capture :")
	g, _ := grok.New(grok.Config{})
	values, _ := g.ParseString("%{COMMONAPACHELOG}", `127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`)
	for k, v := range values {
		fmt.Printf("%+15s: %s\n", k, v)
	}

	fmt.Println("\n# Named Capture :")
	g, _ = grok.New(grok.Config{NamedCapturesOnly: true})
	values, _ = g.ParseString("%{COMMONAPACHELOG}", `127.0.0.1 - - [23/Apr/2014:22:58:32 +0200] "GET /index.php HTTP/1.1" 404 207`)
	for k, v := range values {
		fmt.Printf("%+15s: %s\n", k, v)
	}

	fmt.Println("\n# Add custom patterns :")
	// We add 3 patterns to our Grok instance, to structure an IRC message
	g, _ = grok.New(grok.Config{
		NamedCapturesOnly: true,
		Patterns: map[string]string{
			"IRCUSER": `\A@(\w+)`,
			"IRCBODY": `.*`,
			"IRCMSG":  `%{IRCUSER:user} .* : %{IRCBODY:message}`,
		},
	})
	values, _ = g.ParseString("%{IRCMSG}", `@vjeantet said : Hello !`)
	for k, v := range values {
		fmt.Printf("%+15s: %s\n", k, v)
	}
}
