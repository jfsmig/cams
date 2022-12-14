module github.com/jfsmig/cams/go

go 1.18

require (
	github.com/aler9/gortsplib v0.0.0-20220724100730-2c8889602c59
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/jfsmig/go-bags v0.2.0
	github.com/juju/errors v0.0.0-20220331221717-b38fca44723b
	github.com/rs/zerolog v1.26.1
	github.com/spf13/cobra v1.4.0
	github.com/use-go/onvif v0.0.9
	go.nanomsg.org/mangos/v3 v3.4.2
	google.golang.org/grpc v1.46.0
	google.golang.org/protobuf v1.28.0
)

require (
	github.com/beevik/etree v1.1.0 // indirect
	github.com/elgs/gostrgen v0.0.0-20161222160715-9d61ae07eeae // indirect
	github.com/gofrs/uuid v3.2.0+incompatible // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.9 // indirect
	github.com/pion/rtp v1.7.13 // indirect
	github.com/pion/sdp/v3 v3.0.5 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/net v0.0.0-20220526153639-5463443f8c37 // indirect
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
)

replace github.com/use-go/onvif => github.com/jfsmig/onvif v0.0.2-0.20221212195031-7375d6c78ab1

//replace github.com/aler9/gortsplib => github.com/jfsmig/gortsplib v0.0.0-20220724100730-2c8889602c59
