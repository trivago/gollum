# Deprecated producers

Here you find a collection of deprecated producers which reached the EOL (End of Life) or replaced by a new one.
If you want to use these deprecated producers pls active them explicitly by your `contrib_loader.go`:

```golang
import (
	_ "github.com/trivago/gollum/contrib/deprecated/producer"
)
```

## S3 Producer

### Example config

Console to S3:

```yaml
StdIn:
    Type: consumer.Console
    Streams: console

S3Out:
    Type: deprecated.producer.S3
    Region: eu-west-1

    Credential:
        Type: shared
        File: /Users/MYUSER/.aws/credentials
        Profile: default

    Streams: "*"
    StreamMapping:
        "console": "my-s3-bucket/subfolder/"

    SendTimeframeMs: 1000

    LocalPath: /tmp/aws
    UploadOnShutdown: true
```