// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

/**
 * gRPC codec for the communication toward the hub.
 */
package proto

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative hub.proto
