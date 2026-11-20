package world

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/crypto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/netx"
)

// Each synthetic controller owns a private post-login TCP stream. It uses
// normal world framing and AES, with credentials and account state absent.
type botWire struct {
	client, world net.Conn
	send, receive *crypto.AESCTR
	frames        netx.Frames
}

func openBotWire(listener *net.TCPListener) (*botWire, error) {
	client, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		return nil, err
	}
	world, err := listener.AcceptTCP()
	if err != nil {
		client.Close()
		return nil, err
	}
	var key, iv [16]byte
	cryptorand.Read(key[:])
	cryptorand.Read(iv[:])
	return &botWire{client: client, world: world, send: crypto.NewAESCTR(key, iv), receive: crypto.NewAESCTR(key, iv)}, nil
}

func (w *botWire) close() {
	w.client.Close()
	w.world.Close()
}

func (w *botWire) request(payload []byte) ([]byte, error) {
	frame := binary.LittleEndian.AppendUint16(nil, uint16(len(payload)+2))
	frame = append(frame, w.send.Transform(payload)...)
	deadline := time.Now().Add(time.Second)
	if err := w.client.SetWriteDeadline(deadline); err != nil {
		return nil, err
	}
	if err := w.world.SetReadDeadline(deadline); err != nil {
		return nil, err
	}
	if _, err := w.client.Write(frame); err != nil {
		return nil, err
	}
	var buf [64]byte
	for {
		n, err := w.world.Read(buf[:])
		if err != nil {
			return nil, err
		}
		list, ok := w.frames.Push(buf[:n])
		if !ok || len(list) > 1 {
			return nil, fmt.Errorf("invalid synthetic world frame")
		}
		if len(list) == 1 {
			return w.receive.Transform(list[0][2:]), nil
		}
	}
}
