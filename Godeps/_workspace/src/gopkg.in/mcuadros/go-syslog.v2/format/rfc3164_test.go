package format

import (
	. "launchpad.net/gocheck"
)

func (s *FormatSuite) TestRFC3164_SingleSplit(c *C) {
	f := RFC3164{}
	c.Assert(f.GetSplitFunc(), IsNil)
}
