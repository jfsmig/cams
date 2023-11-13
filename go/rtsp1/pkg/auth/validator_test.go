package auth

import (
	base2 "github.com/jfsmig/cams/go/rtsp1/pkg/base"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatorErrors(t *testing.T) {
	for _, ca := range []struct {
		name string
		hv   base2.HeaderValue
		err  string
	}{
		{
			"invalid auth",
			base2.HeaderValue{`Invalid`},
			"invalid authorization header",
		},
		{
			"digest missing realm",
			base2.HeaderValue{`Digest `},
			"realm is missing",
		},
		{
			"digest missing nonce",
			base2.HeaderValue{`Digest realm=123`},
			"nonce is missing",
		},
		{
			"digest missing username",
			base2.HeaderValue{`Digest realm=123,nonce=123`},
			"username is missing",
		},
		{
			"digest missing uri",
			base2.HeaderValue{`Digest realm=123,nonce=123,username=123`},
			"uri is missing",
		},
		{
			"digest missing response",
			base2.HeaderValue{`Digest realm=123,nonce=123,username=123,uri=123`},
			"response is missing",
		},
		{
			"digest wrong nonce",
			base2.HeaderValue{`Digest realm=123,nonce=123,username=123,uri=123,response=123`},
			"wrong nonce",
		},
		{
			"digest wrong realm",
			base2.HeaderValue{`Digest realm=123,nonce=abcde,username=123,uri=123,response=123`},
			"wrong realm",
		},
	} {
		t.Run(ca.name, func(t *testing.T) {
			va := NewValidator("myuser", "mypass", nil)
			va.nonce = "abcde"
			err := va.ValidateRequest(&base2.Request{
				Method: base2.Describe,
				URL:    nil,
				Header: base2.Header{
					"Authorization": ca.hv,
				},
			}, nil)
			require.EqualError(t, err, ca.err)
		})
	}
}
