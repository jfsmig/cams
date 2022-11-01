// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package utils

import (
	"bytes"
	"github.com/jfsmig/cams/api/pb"
)

func MediaEncode(user, id string, frameType pb.MediaFrameType, b []byte) []byte {
	userLen := len(user)
	idLen := len(id)
	msgSize := userLen + 1 + idLen + 1 + len(b)
	msg := make([]byte, msgSize, msgSize)

	copy(msg[0:], []byte(user))
	msg[userLen] = 0
	copy(msg[userLen+1:], []byte(id))
	msg[userLen+1+idLen] = 0
	copy(msg[userLen+1+idLen+1:], b)

	return msg
}

func MediaDecode(msg []byte) *pb.MediaFrame {
	// Extract the identifiers of the owner and the stream itself.
	offsetUser := 0
	offsetID := bytes.IndexByte(msg[offsetUser:], 0)
	if offsetID < 0 {
		panic("invalid internal msg (id)")
	}
	offsetID++
	offsetFrame := bytes.IndexByte(msg[offsetID:], 0)
	if offsetFrame < 0 {
		panic("invalid internal msg (frame)")
	}
	offsetFrame++

	// Copy the message
	frame := &pb.MediaFrame{}
	frame.Id = &pb.StreamId{
		User:   string(msg[offsetUser : offsetID-1]),
		Stream: string(msg[offsetID : offsetFrame-1]),
	}
	frame.Payload = msg

	return frame
}
