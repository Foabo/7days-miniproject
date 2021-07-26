package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn) 	// buf是conn的缓冲
	return &GobCodec{
		conn: conn,                 //	 这个conn还可以看成连接套接字的封装
		buf:  buf,                  //	conn的缓冲
		dec:  gob.NewDecoder(conn), // conn decode 到其他变量
		enc:  gob.NewEncoder(buf),  // 数据encode到buf
	}
}

// decode conn 为 Header
func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

// decode conn 为 body
func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

// 将 h 和 body encode 到 buf
func (c *GobCodec) Write(h *Header, body interface{}) (err error) {
	// 调用 buffer.Flush() 来将 buffer 中的全部内容写入到 conn 中
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()
	// 连续两个encode是组合在一起了，拼接在一块
	// 在buf中就有了h和body两个按顺序的存储
	if err := c.enc.Encode(h); err != nil {
		log.Println("rpc codec: gob error encoding header:", err)
		return err
	}

	if err := c.enc.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding body:", err)
		return err
	}

	return nil

}

func (c *GobCodec) Close() error {
	return c.conn.Close()
}
