package http

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/d7561985/tel/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// want to checl if response will write to log via mw
const testString = "Hello World"

// want to check if it write to log via mw
const postContent = "XXX"

type Suite struct {
	suite.Suite

	tel   tel.Telemetry
	close func()

	buf *bytes.Buffer
}

func (s *Suite) SetupSuite() {
	c := tel.DefaultDebugConfig()
	c.LogLevel = "debug"
	c.OtelConfig.Enable = false

	s.tel, s.close = tel.New(context.Background(), c)
	s.buf = tel.SetLogOutput(&s.tel)
}

func (s *Suite) TearDownSuite() {
	s.close()
}

func (s *Suite) TearDownTest() {
	s.buf.Reset()
}

func TestHTTP(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestFilter() {
	var ok bool

	mw := NewServeMux(WithTel(&s.tel))
	mw.Handle("/ws", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok = true
	}))
	mw.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok = true
	}))

	ss := httptest.NewServer(mw)
	defer ss.Close()

	s.Run("health", func() {
		ok = false

		_, err := ss.Client().Get(ss.URL + "/health")
		s.NoError(err)

		s.True(ok)
		s.Empty(s.buf.Bytes())
	})

	s.Run("websocket", func() {
		ok = false

		b := bytes.NewReader(nil)
		r := httptest.NewRequest(http.MethodGet, ss.URL+"/ws", b)
		r.Header.Set("Upgrade", "websocket")
		r.RequestURI = ""

		_, err := ss.Client().Do(r)
		s.NoError(err)

		s.True(ok)
		s.Empty(s.buf.Bytes())
	})
}

func (s *Suite) TestHttpServerMiddlewareAll() {
	// key value helps check if our middleware not damage already existent context with own values
	type key struct{}

	handler := http.Handler(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(testString))

		assert.NotNil(s.T(), request.Context().Value(key{}))
		fmt.Println(request.Context().Value(key{}))
	}))

	mw := NewServeMux(WithTel(&s.tel))
	mw.Handle("/test", func(next http.Handler) http.Handler {
		fn := func(writer http.ResponseWriter, request *http.Request) {
			ctx := context.WithValue(request.Context(), key{}, "*****")
			next.ServeHTTP(writer, request.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}(handler))

	ss := httptest.NewServer(mw)
	defer ss.Close()

	res, err := ss.Client().Post(ss.URL+"/test", "", bytes.NewBufferString(postContent))
	s.NoError(err)

	v, err := ioutil.ReadAll(res.Body)
	s.NoError(err)
	s.Contains(string(v), testString)
	s.Contains(s.buf.String(), testString)
	s.Contains(s.buf.String(), postContent)

	fmt.Println(s.buf.String())
}
