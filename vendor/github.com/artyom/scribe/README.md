# Facebook Scribe for Go

This package is built automatically from [Thrift repository][1] at rev
`b5948eb378db07906594813b3e170b64d4352487` and [Scribe][2] at rev
`7359a099ed64278849909b363b7f2f0ba059808d`.

	thrift \
		--gen go:package_prefix=github.com/artyom/ \
		if/scribe.thrift

[1]: https://git-wip-us.apache.org/repos/asf/thrift.git
[2]: https://github.com/facebook/scribe
