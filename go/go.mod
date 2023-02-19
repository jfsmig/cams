module github.com/jfsmig/cams/go

go 1.19

require (
	github.com/aler9/gortsplib/v2 v2.0.5
	github.com/google/uuid v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/jfsmig/go-bags v0.2.0
	github.com/jfsmig/onvif v1.1.0
	github.com/jfsmig/streaming v0.0.0-20230219132329-a4d51af4cba7
	github.com/juju/errors v1.0.0
	github.com/pion/rtcp v1.2.9
	github.com/pion/rtp v1.7.13
	github.com/pion/sdp/v3 v3.0.6
	github.com/rs/zerolog v1.29.0
	github.com/spf13/cobra v1.6.1
	github.com/stretchr/testify v1.7.1
	golang.org/x/sync v0.1.0
	google.golang.org/grpc v1.46.0
	google.golang.org/protobuf v1.28.0
)

require (
	github.com/beevik/etree v1.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/elgs/gostrgen v0.0.0-20161222160715-9d61ae07eeae // indirect
	github.com/gofrs/uuid v3.2.0+incompatible // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

//replace github.com/aler9/gortsplib/v2 => github.com/jfsmig/gortsplib/v2 v0.0.0-20220724100730-2c8889602c59
