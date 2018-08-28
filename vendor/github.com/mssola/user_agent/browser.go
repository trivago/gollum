// Copyright (C) 2012-2014 Miquel Sabaté Solà <mikisabate@gmail.com>
// This file is licensed under the MIT license.
// See the LICENSE file.

package user_agent

import (
	"regexp"
	"strings"
)

// A struct containing all the information that we might be
// interested from the browser.
type Browser struct {
	// The name of the browser's engine.
	Engine string

	// The version of the browser's engine.
	EngineVersion string

	// The name of the browser.
	Name string

	// The version of the browser.
	Version string
}

// Extract all the information that we can get from the User-Agent string
// about the browser and update the receiver with this information.
//
// The function receives just one argument "sections", that contains the
// sections from the User-Agent string after being parsed.
func (p *UserAgent) detectBrowser(sections []section) {
	slen := len(sections)

	if sections[0].name == "Opera" {
		p.mozilla = ""
		p.browser.Name = "Opera"
		p.browser.Version = sections[0].version
		p.browser.Engine = "Presto"
		if slen > 1 {
			p.browser.EngineVersion = sections[1].version
		}
	} else if slen > 1 {
		engine := sections[1]
		p.browser.Engine = engine.name
		p.browser.EngineVersion = engine.version
		if slen > 2 {
			p.browser.Version = sections[2].version
			if engine.name == "AppleWebKit" {
				if sections[slen-1].name == "OPR" {
					p.browser.Name = "Opera"
					p.browser.Version = sections[slen-1].version
				} else if sections[2].name == "Chrome" {
					p.browser.Name = "Chrome"
				} else {
					p.browser.Name = "Safari"
				}
			} else if engine.name == "Gecko" {
				name := sections[2].name
				if name == "MRA" && slen > 4 {
					name = sections[4].name
					p.browser.Version = sections[4].version
				}
				p.browser.Name = name
			} else if engine.name == "like" && sections[2].name == "Gecko" {
				// This is the new user agent from Internet Explorer 11.
				p.browser.Engine = "Trident"
				p.browser.Name = "Internet Explorer"
				reg, _ := regexp.Compile("^rv:(.+)$")
				for _, c := range sections[0].comment {
					version := reg.FindStringSubmatch(c)
					if len(version) > 0 {
						p.browser.Version = version[1]
						return
					}
				}
				p.browser.Version = ""
			}
		}
	} else if slen == 1 && len(sections[0].comment) > 1 {
		comment := sections[0].comment
		if comment[0] == "compatible" && strings.HasPrefix(comment[1], "MSIE") {
			p.browser.Engine = "Trident"
			p.browser.Name = "Internet Explorer"
			// The MSIE version may be reported as the compatibility version.
			// For IE 8 through 10, the Trident token is more accurate.
			// http://msdn.microsoft.com/en-us/library/ie/ms537503(v=vs.85).aspx#VerToken
			for _, v := range comment {
				if strings.HasPrefix(v, "Trident/") {
					switch v[8:] {
					case "4.0":
						p.browser.Version = "8.0"
					case "5.0":
						p.browser.Version = "9.0"
					case "6.0":
						p.browser.Version = "10.0"
					}
					break
				}
			}
			// If the Trident token is not provided, fall back to MSIE token.
			if p.browser.Version == "" {
				p.browser.Version = strings.TrimSpace(comment[1][4:])
			}
		}
	}
}

// Returns two strings. The first string is the name of the engine and the
// second one is the version of the engine.
func (p *UserAgent) Engine() (string, string) {
	return p.browser.Engine, p.browser.EngineVersion
}

// Returns two strings. The first string is the name of the browser and the
// second one is the version of the browser.
func (p *UserAgent) Browser() (string, string) {
	return p.browser.Name, p.browser.Version
}
