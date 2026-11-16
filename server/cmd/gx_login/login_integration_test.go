//go:build integration

// Run with: go test -tags integration ./cmd/gx_login
// Uses a disposable SQLite runtime copy and the admin1 test account.
package main

import (
	"crypto/rand"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/authwire"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	gxsession "github.com/Spelunkern/Shaiya-GX/internal/session"
	"github.com/Spelunkern/Shaiya-GX/internal/transport"
	"github.com/Spelunkern/Shaiya-GX/server/internal/admin"
	"io"
	"math/big"
	"net"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/crypto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"github.com/Spelunkern/Shaiya-GX/server/internal/session"
)

func readFrame(t *testing.T, conn net.Conn) []byte {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	head := make([]byte, 2)
	if _, err := io.ReadFull(conn, head); err != nil {
		t.Fatal(err)
	}
	frame := make([]byte, int(head[0])|int(head[1])<<8)
	copy(frame, head)
	if _, err := io.ReadFull(conn, frame[2:]); err != nil {
		t.Fatal(err)
	}
	return frame
}

func le(b []byte) *big.Int { r := slices.Clone(b); slices.Reverse(r); return new(big.Int).SetBytes(r) }

func TestLoginAgainstSQLite(t *testing.T) {
	for _, extended := range []bool{false, true} {
		t.Run(fmt.Sprint(extended), func(t *testing.T) { testLoginSQLite(t, extended, false) })
	}
	t.Run("tls_long_real_client", func(t *testing.T) { testLoginSQLite(t, true, true) })
}
func testLoginSQLite(t *testing.T, extended, secure bool) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("RAPTURE_DB_PATH", filepath.Join(t.TempDir(), "runtime.sqlite"))
	t.Setenv("RAPTURE_DB_SEED", filepath.Join(root, "data", "rapture.sqlite"))
	t.Chdir(t.TempDir())
	pool, err := db.Pool(db.UserData)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err = pool.Exec(`INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(990010,'sqlite_login','localtest',0,0,0,0,0,'N',0)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(`DELETE FROM PS_UserData_UserLoginStatus WHERE UserUID=990010`)
		pool.Exec(`DELETE FROM PS_UserData_Users_Master WHERE UserUID=990010`)
		pool.Exec(`DELETE FROM GX_SessionHandoff WHERE UserUID=990010`)
	})
	logger, err := logx.New("gx_login_test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { logger.Close() })
	password := "localtest"
	if extended {
		password = strings.Repeat("long-passphrase-", 7)
		if err := db.SetAccountPassword("sqlite_login", password); err != nil {
			t.Fatal(err)
		}
	}
	if secure {
		certs := filepath.Join(t.TempDir(), "certificates")
		if err := admin.Run([]string{"tls-init", "-out", certs}); err != nil {
			t.Fatal(err)
		}
		t.Setenv("RAPTURE_TLS_CERT", filepath.Join(certs, "server-cert.pem"))
		t.Setenv("RAPTURE_TLS_KEY", filepath.Join(certs, "server-key.pem"))
		t.Setenv("GX_NETWORK_CONFIG", filepath.Join(certs, "client", "network.json"))
	}
	listener, err := transport.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		conn, err := listener.Accept()
		if err == nil {
			handle(conn, logger.Tagged("test"))
			conn.Close()
		}
	}()
	if secure {
		client, err := gxsession.Dial(listener.Addr().String(), "sqlite_login", password)
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()
		if _, err := client.SelectServer(client.Servers[0].ID, proto.ClientVersion); err != nil {
			t.Fatal(err)
		}
		if _, err := session.Consume(client.Result.UserUID, client.Result.Token); err != nil {
			t.Fatal(err)
		}
		select {
		case <-finished:
		case <-time.After(5 * time.Second):
			t.Fatal("TLS handler did not finish")
		}
		return
	}
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Close()
		select {
		case <-finished:
		case <-time.After(5 * time.Second):
			t.Error("login handler did not finish")
		}
	})

	hello := readFrame(t, conn)
	if op, _ := packet.U16(hello[2:]); packet.Opcode(op) != packet.LoginHandshake {
		t.Fatalf("first opcode 0x%04x", op)
	}
	e, n := le(hello[7:71]), le(hello[71:199])
	secret := make([]byte, 64)
	rand.Read(secret)
	cipher := new(big.Int).Exp(le(secret), e, n).Bytes()
	slices.Reverse(cipher)
	w := packet.New(packet.LoginHandshake)
	w.U8(uint8(len(cipher)))
	w.Bytes(cipher)
	conn.Write(w.MustFinish())

	material := crypto.DeriveLoginAESMaterial(crypto.DecryptLoginRSA(cipher))
	var key, iv [16]byte
	copy(key[:], material[:16])
	copy(iv[:], material[16:])
	send, recv := crypto.NewAESCTR(key, iv), crypto.NewAESCTR(key, iv)

	login := make([]byte, 50)
	copy(login[0:], []byte{0x02, 0xA1})
	copy(login[2:], "sqlite_login")
	copy(login[34:], password)
	if extended {
		var err error
		login, err = authwire.Encode("sqlite_login", password)
		if err != nil {
			t.Fatal(err)
		}
	}
	frame := append([]byte{byte(len(login) + 2), 0}, send.Transform(login)...)
	conn.Write(frame)

	reply := readFrame(t, conn)
	body := recv.Transform(reply[2:])
	if op, _ := packet.U16(body); packet.Opcode(op) != packet.LoginRequest {
		t.Fatalf("reply opcode 0x%04x", op)
	}
	if body[2] != 0 {
		t.Fatalf("login result %d", body[2])
	}
	uid, _ := packet.U32(body[3:])
	t.Logf("login ok user_uid=%d permission=%d", uid, body[7])
	list := recv.Transform(readFrame(t, conn)[2:])
	if op, _ := packet.U16(list); packet.Opcode(op) != packet.ServerList {
		t.Fatalf("expected server list, got 0x%04x", op)
	}
	var identity [16]byte
	copy(identity[:], body[8:24])
	if _, err := session.Consume(uid, identity); err != nil {
		t.Fatalf("handoff missing: %v", err)
	}
}
