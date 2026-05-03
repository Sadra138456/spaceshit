package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"

	"github.com/hashicorp/yamux"
)

func handleSOCKS5(local net.Conn, session *yamux.Session) {
	defer local.Close()

	buf := make([]byte, 262)

	// greeting
	n, err := local.Read(buf)
	if err != nil || n < 2 {
		return
	}

	local.Write([]byte{0x05, 0x00})

	// request
	n, err = local.Read(buf)
	if err != nil {
		return
	}

	atyp := buf[3]

	stream, err := session.Open()
	if err != nil {
		return
	}
	defer stream.Close()

	switch atyp {

	case 0x01: // IPv4
		addr := buf[4:8]
		port := buf[8:10]

		stream.Write([]byte{0x01})
		stream.Write(addr)
		stream.Write(port)

	case 0x03: // domain
		length := int(buf[4])
		domain := buf[5 : 5+length]
		port := buf[5+length : 7+length]

		stream.Write([]byte{0x03})
		stream.Write([]byte{byte(length)})
		stream.Write(domain)
		stream.Write(port)

	default:
		return
	}

	resp := make([]byte, 2)
	if _, err := io.ReadFull(stream, resp); err != nil {
		return
	}

	if resp[1] != 0x00 {
		local.Write([]byte{0x05, 0x01})
		return
	}

	local.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	go io.Copy(stream, local)
	io.Copy(local, stream)
}
