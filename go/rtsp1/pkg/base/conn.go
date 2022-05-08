// Package conn contains a RTSP connection implementation.
package base

import (
	"bufio"
	"io"
)

const (
	readBufferSize = 4096
)

// Conn is a RTSP connection.
type Conn struct {
	w   io.Writer
	br  *bufio.Reader
	req Request
	res Response
	fr  InterleavedFrame
}

// NewConn allocates a Conn.
func NewConn(rw io.ReadWriter) *Conn {
	return &Conn{
		w:  rw,
		br: bufio.NewReaderSize(rw, readBufferSize),
	}
}

// ReadRequest reads a Request.
func (c *Conn) ReadRequest() (*Request, error) {
	err := c.req.Read(c.br)
	return &c.req, err
}

// ReadResponse reads a Response.
func (c *Conn) ReadResponse() (*Response, error) {
	err := c.res.Read(c.br)
	return &c.res, err
}

// ReadInterleavedFrame reads a InterleavedFrame.
func (c *Conn) ReadInterleavedFrame() (*InterleavedFrame, error) {
	err := c.fr.Read(c.br)
	return &c.fr, err
}

// ReadInterleavedFrameOrRequest reads an InterleavedFrame or a Request.
func (c *Conn) ReadInterleavedFrameOrRequest() (interface{}, error) {
	b, err := c.br.ReadByte()
	if err != nil {
		return nil, err
	}
	c.br.UnreadByte()

	if b == InterleavedFrameMagicByte {
		return c.ReadInterleavedFrame()
	}

	return c.ReadRequest()
}

// ReadInterleavedFrameOrResponse reads an InterleavedFrame or a Response.
func (c *Conn) ReadInterleavedFrameOrResponse() (interface{}, error) {
	b, err := c.br.ReadByte()
	if err != nil {
		return nil, err
	}
	c.br.UnreadByte()

	if b == InterleavedFrameMagicByte {
		return c.ReadInterleavedFrame()
	}

	return c.ReadResponse()
}

// ReadRequestIgnoreFrames reads a Request and ignores frames in between.
func (c *Conn) ReadRequestIgnoreFrames() (*Request, error) {
	for {
		recv, err := c.ReadInterleavedFrameOrRequest()
		if err != nil {
			return nil, err
		}

		if req, ok := recv.(*Request); ok {
			return req, nil
		}
	}
}

// ReadResponseIgnoreFrames reads a Response and ignores frames in between.
func (c *Conn) ReadResponseIgnoreFrames() (*Response, error) {
	for {
		recv, err := c.ReadInterleavedFrameOrResponse()
		if err != nil {
			return nil, err
		}

		if res, ok := recv.(*Response); ok {
			return res, nil
		}
	}
}

// WriteRequest writes a request.
func (c *Conn) WriteRequest(req *Request) error {
	buf, _ := req.Marshal()
	_, err := c.w.Write(buf)
	return err
}

// WriteResponse writes a response.
func (c *Conn) WriteResponse(res *Response) error {
	buf, _ := res.Marshal()
	_, err := c.w.Write(buf)
	return err
}

// WriteInterleavedFrame writes an interleaved frame.
func (c *Conn) WriteInterleavedFrame(fr *InterleavedFrame, buf []byte) error {
	n, _ := fr.MarshalTo(buf)
	_, err := c.w.Write(buf[:n])
	return err
}
