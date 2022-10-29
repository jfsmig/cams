// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package utils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/juju/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"time"
)

type ServerConfig struct {
	ListenAddr string
	PathCrt    string
	PathKey    string
}

func (srv *ServerConfig) ServeTLS() (*grpc.Server, error) {
	if len(srv.PathCrt) <= 0 {
		return nil, errors.NotValidf("invalid TLS/x509 certificate path [%s]", srv.PathCrt)
	}
	if len(srv.PathKey) <= 0 {
		return nil, errors.NotValidf("invalid TLS/x509 key path [%s]", srv.PathKey)
	}
	var certBytes, keyBytes []byte
	var err error

	Logger.Info().Str("key", srv.PathKey).Str("crt", srv.PathCrt).Msg("TLS config")

	if certBytes, err = ioutil.ReadFile(srv.PathCrt); err != nil {
		return nil, errors.Annotate(err, "certificate file error")
	}
	if keyBytes, err = ioutil.ReadFile(srv.PathKey); err != nil {
		return nil, errors.Annotate(err, "key file error")
	}

	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM(certBytes)
	if !ok {
		return nil, errors.New("invalid certificates")
	}

	cert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		return nil, errors.Annotate(err, "x509 key pair error")
	}

	return grpc.NewServer(
		grpc.Creds(credentials.NewServerTLSFromCert(&cert)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			//grpc_prometheus.UnaryServerInterceptor,
			NewUnaryServerInterceptorZerolog())),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			//grpc_prometheus.StreamServerInterceptor,
			NewStreamServerInterceptorZerolog()))), nil
}

func DialGrpc(ctx context.Context, endpoint string) (*grpc.ClientConn, error) {
	config := &tls.Config{
		InsecureSkipVerify: true,
	}

	options := []grpc_retry.CallOption{
		grpc_retry.WithCodes(codes.Unavailable, codes.Unimplemented),
		grpc_retry.WithBackoff(
			grpc_retry.BackoffExponentialWithJitter(250*time.Millisecond, 0.1),
		),
		grpc_retry.WithMax(5),
		grpc_retry.WithPerRetryTimeout(1 * time.Second),
	}

	return grpc.DialContext(ctx, endpoint,
		grpc.WithTransportCredentials(credentials.NewTLS(config)),
		grpc.WithUnaryInterceptor(
			grpc_middleware.ChainUnaryClient(
				//grpc_prometheus.UnaryClientInterceptor,
				grpc_retry.UnaryClientInterceptor(options...),
			)),
		grpc.WithStreamInterceptor(
			grpc_middleware.ChainStreamClient(
				//grpc_prometheus.StreamClientInterceptor,
				grpc_retry.StreamClientInterceptor(options...),
			)),
	)
}
