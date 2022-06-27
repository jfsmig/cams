// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"github.com/jfsmig/go-bags"
	"github.com/juju/errors"
)

type localStreamSwitch struct {
	sources bags.SortedObj[string, StreamSource]
}

func NewStreamPlayer() StreamPlayer {
	return &localStreamSwitch{}
}

func (ls *localStreamSwitch) Run(ctx context.Context) error {
	return errors.NotImplemented
}

func (ls *localStreamSwitch) Register(src StreamSource) error {
	return errors.NotImplemented
}
