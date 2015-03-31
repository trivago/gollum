package format

import (
	. "launchpad.net/gocheck"
)

func (s *FormatSuite) TestRFC5424_SingleSplit(c *C) {
	f := RFC5424{}
	c.Assert(f.GetSplitFunc(), IsNil)
}
