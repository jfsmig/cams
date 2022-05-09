package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"text/template"
)

var mainTemplate = `// Code generated : DO NOT EDIT.

// Copyright (c) 2022 Jean-Francois Smigielski
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package {{.Package}}

import (
	"encoding/xml"
	"io/ioutil"
	"github.com/juju/errors"
	goonvif "github.com/use-go/onvif"
	"github.com/use-go/onvif/media"
)

func call_{{.BareTypeRequest}}_parse_{{.BareTypeReply}}(dev *goonvif.Device, request {{.TypeRequest}}) ({{.TypeReply}}, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			{{.BareTypeReply}} {{.TypeReply}}
		}
	}

	var reply Envelope

	if httpReply, err := dev.CallMethod(request); err != nil {
		return reply.Body.{{.BareTypeReply}}, errors.Trace(err)
	} else {
		Logger.Debug().
			Str("msg", httpReply.Status).
			Int("status", httpReply.StatusCode).
			Str("rpc", "{{.BareTypeReply}}").
			Msg("RPC")

		// FIXME(jfs): Get rid of this buffering
		b, err := ioutil.ReadAll(httpReply.Body)
		if err != nil {
			return reply.Body.{{.BareTypeReply}}, errors.Trace(err)
		}
		if err = xml.Unmarshal(b, &reply); err != nil {
			return reply.Body.{{.BareTypeReply}}, errors.Trace(err)
		} 
		return reply.Body.{{.BareTypeReply}}, nil
	}
}

`

type parserEnv struct {
	Path            string
	Package         string
	TypeReply       string
	TypeRequest     string
	BareTypeReply   string
	BareTypeRequest string
}

func lastToken(s string) string {
	tokens := strings.Split(s, ".")
	return tokens[len(tokens)-1]
}

func main() {
	flag.Parse()
	env := parserEnv{
		Path:        flag.Arg(0),
		Package:     flag.Arg(1),
		TypeRequest: flag.Arg(2),
		TypeReply:   flag.Arg(3),
	}

	env.BareTypeReply = lastToken(env.TypeReply)
	env.BareTypeRequest = lastToken(env.TypeRequest)

	body, err := template.New("body").Parse(mainTemplate)
	if err != nil {
		log.Fatalln(err)
	}

	fout, err := os.Create(env.Path)
	if err != nil {
		log.Fatalln(err)
	}
	defer fout.Close()

	err = body.Execute(fout, &env)
	if err != nil {
		log.Fatalln(err)
	}
}
