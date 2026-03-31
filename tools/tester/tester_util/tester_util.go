package tester_util

import (
	"github.com/Iori372552686/GoOne/lib/api/logger"
	"github.com/Iori372552686/GoOne/lib/api/sharedstruct"
	"github.com/golang/protobuf/proto"
	"io"
	"net"
)

//todo: 不应该有uid
func SendCmd(conn net.Conn, uid uint64, cmd uint32, req proto.Message) error {
	out, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	// header
	header := sharedstruct.CSPacketHeader{}
	header.Uid = uid
	header.Cmd = cmd
	header.BodyLen = uint32(len(out))
	conn.Write(header.ToBytes())
	// body
	conn.Write(out)

	return nil
}

func WaitTillCmd(conn net.Conn, cmd uint32, rsp proto.Message) error {
	header := sharedstruct.CSPacketHeader{}
	headerBuf := make([]byte, sharedstruct.ByteLenOfCSPacketHeader())

	for header.Cmd != cmd {
		_, err := io.ReadFull(conn, headerBuf)
		if err != nil {
			return err
		}

		header.From(headerBuf)

		body := make([]byte, header.BodyLen)
		_, err = io.ReadFull(conn, body)
		if err != nil {
			return err
		}

		logger.Debugf("received: %#v", header)

		if header.Cmd == cmd {
			err = proto.Unmarshal(body, rsp)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func WaitTillAnyCmd(conn net.Conn, rsps map[uint32]proto.Message) (uint32, error) {
	header := sharedstruct.CSPacketHeader{}
	headerBuf := make([]byte, sharedstruct.ByteLenOfCSPacketHeader())

	for {
		_, err := io.ReadFull(conn, headerBuf)
		if err != nil {
			return 0, err
		}

		header.From(headerBuf)
		body := make([]byte, header.BodyLen)
		_, err = io.ReadFull(conn, body)
		if err != nil {
			return 0, err
		}

		logger.Debugf("received: %#v", header)
		rsp := rsps[header.Cmd]
		if rsp == nil {
			continue
		}
		if err = proto.Unmarshal(body, rsp); err != nil {
			return 0, err
		}
		return header.Cmd, nil
	}
}
