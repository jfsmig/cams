package auth

import (
	"fmt"
	base2 "github.com/jfsmig/streaming/rtsp1/pkg/base"
	headers2 "github.com/jfsmig/streaming/rtsp1/pkg/headers"
	"strings"
)

// Sender allows to generate credentials for a Validator.
type Sender struct {
	user   string
	pass   string
	method headers2.AuthMethod
	realm  string
	nonce  string
}

// NewSender allocates a Sender with the WWW-Authenticate header provided by
// a Validator and a set of credentials.
func NewSender(v base2.HeaderValue, user string, pass string) (*Sender, error) {
	// prefer digest
	if v0 := func() string {
		for _, vi := range v {
			if strings.HasPrefix(vi, "Digest") {
				return vi
			}
		}
		return ""
	}(); v0 != "" {
		var auth headers2.Authenticate
		err := auth.Unmarshal(base2.HeaderValue{v0})
		if err != nil {
			return nil, err
		}

		if auth.Realm == nil {
			return nil, fmt.Errorf("realm is missing")
		}

		if auth.Nonce == nil {
			return nil, fmt.Errorf("nonce is missing")
		}

		return &Sender{
			user:   user,
			pass:   pass,
			method: headers2.AuthDigest,
			realm:  *auth.Realm,
			nonce:  *auth.Nonce,
		}, nil
	}

	if v0 := func() string {
		for _, vi := range v {
			if strings.HasPrefix(vi, "Basic") {
				return vi
			}
		}
		return ""
	}(); v0 != "" {
		var auth headers2.Authenticate
		err := auth.Unmarshal(base2.HeaderValue{v0})
		if err != nil {
			return nil, err
		}

		if auth.Realm == nil {
			return nil, fmt.Errorf("realm is missing")
		}

		return &Sender{
			user:   user,
			pass:   pass,
			method: headers2.AuthBasic,
			realm:  *auth.Realm,
		}, nil
	}

	return nil, fmt.Errorf("no authentication methods available")
}

// AddAuthorization adds the Authorization header to a Request.
func (se *Sender) AddAuthorization(req *base2.Request) {
	urStr := req.URL.CloneWithoutCredentials().String()

	h := headers2.Authorization{
		Method: se.method,
	}

	switch se.method {
	case headers2.AuthBasic:
		h.BasicUser = se.user
		h.BasicPass = se.pass

	default: // headers.AuthDigest
		response := md5Hex(md5Hex(se.user+":"+se.realm+":"+se.pass) + ":" +
			se.nonce + ":" + md5Hex(string(req.Method)+":"+urStr))

		h.DigestValues = headers2.Authenticate{
			Method:   headers2.AuthDigest,
			Username: &se.user,
			Realm:    &se.realm,
			Nonce:    &se.nonce,
			URI:      &urStr,
			Response: &response,
		}
	}

	if req.Header == nil {
		req.Header = make(base2.Header)
	}

	req.Header["Authorization"] = h.Marshal()
}
