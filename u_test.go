package tel

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const readLimit = 8192

// U is UDP graylog server
// Yes. it's simple graylog server!!!
// with convenient option to read messages via sync way
// Note! This not support chunks! So readLimit is 8192
type U struct {
	net.PacketConn

	pause string
}

func newU(t *testing.T) *U {
	l, err := net.ListenPacket("udp", "0.0.0.0:0")
	require.NoError(t, err)

	return &U{PacketConn: l}
}

// Pause close connection and save state
func (u *U) Pause() error {
	u.pause = u.LocalAddr().String()
	return u.Close()
}

// Resume listening
func (u *U) Resume() error {
	if u.pause == "" {
		return fmt.Errorf("nothing to resume")
	}

	l, err := net.ListenPacket("udp", u.pause)
	if err != nil {
		return err
	}

	u.pause, u.PacketConn = "", l

	return nil
}

// readMessages graylog message packed with gzip
func (u *U) readMessages(t *testing.T, n int) bytes.Buffer {
	var inBound bytes.Buffer

	buf := make([]byte, readLimit)

	for i := 0; i < n; i++ {
		err := u.PacketConn.SetReadDeadline(time.Now().Add(time.Second * 5))
		require.NoError(t, err)

		n, _, err := u.PacketConn.ReadFrom(buf)
		assert.NoError(t, err)

		r, err := gzip.NewReader(bytes.NewBuffer(buf[:n]))
		require.NoError(t, err)

		res, err := ioutil.ReadAll(r)
		require.NoError(t, err)

		inBound.Write(res)
	}

	return inBound
}
