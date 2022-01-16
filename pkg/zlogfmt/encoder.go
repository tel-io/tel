package zlogfmt

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-logfmt/logfmt"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var (
	_pool = buffer.NewPool()
	// Get retrieves a buffer from the pool, creating one if necessary.
	Get = _pool.Get
)

type ObjectEncoder struct {
	*logfmt.Encoder
	buf *bytes.Buffer

	// for encoding generic values by reflection
	reflectBuf *buffer.Buffer
	reflectEnc *json.Encoder
}

func New(buf []byte) *ObjectEncoder {
	b := bytes.NewBuffer(buf)
	if b.Len() > 0 {
		b.Write([]byte(" "))
	}

	return &ObjectEncoder{
		buf:     b,
		Encoder: logfmt.NewEncoder(b),
	}
}

func (o *ObjectEncoder) Clone(fields []zapcore.Field) *ObjectEncoder {
	oe := New(o.buf.Bytes())
	for _, field := range fields {
		field.AddTo(oe)
	}

	return oe
}

func (o *ObjectEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) ([]byte, error) {
	if entry.Caller.Defined {
		fields = append(fields, zap.String(CallerKey, entry.Caller.TrimmedPath()))
	}

	if len(entry.Stack) > 0 {
		fields = append(fields, zap.String(StacktraceKey, entry.Stack))
	}

	return o.Clone(
		append(fields, zap.String(MessageKey, entry.Message)),
	).buf.Bytes(), nil
}

func (o *ObjectEncoder) resetReflectBuf() {
	if o.reflectBuf == nil {
		o.reflectBuf = Get()
		o.reflectEnc = json.NewEncoder(o.reflectBuf)

		// For consistency with our custom JSON encoder.
		o.reflectEnc.SetEscapeHTML(false)
	} else {
		o.reflectBuf.Reset()
	}
}

func (o *ObjectEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	//TODO implement me
	panic("implement me")
}

func (o *ObjectEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	_ = o.EndRecord()

	return marshaler.MarshalLogObject(o)
}

func (o ObjectEncoder) AddBinary(key string, value []byte) {
	_ = o.EncodeKeyval(key, base64.StdEncoding.EncodeToString(value))
}

func (o ObjectEncoder) AddByteString(key string, value []byte) {
	_ = o.EncodeKeyval(key, string(value))
}

func (o ObjectEncoder) AddBool(key string, value bool) {
	_ = o.EncodeKeyval(key, fmt.Sprintf("%t", value))
}

func (o ObjectEncoder) AddComplex128(key string, value complex128) {
	_ = o.EncodeKeyval(key, value)
}

func (o ObjectEncoder) AddComplex64(key string, value complex64) {
	_ = o.EncodeKeyval(key, value)
}

func (o ObjectEncoder) AddDuration(key string, value time.Duration) {
	_ = o.EncodeKeyval(key, value.String())
}

func (o ObjectEncoder) AddFloat64(key string, value float64) {
	_ = o.EncodeKeyval(key, value)
}

func (o ObjectEncoder) AddFloat32(key string, value float32) {
	_ = o.EncodeKeyval(key, value)
}

func (o ObjectEncoder) AddInt(key string, value int) {
	_ = o.EncodeKeyval(key, value)
}

func (o ObjectEncoder) AddInt64(key string, value int64) {
	_ = o.EncodeKeyval(key, value)
}

func (o ObjectEncoder) AddInt32(key string, value int32) {
	_ = o.EncodeKeyval(key, value)
}

func (o ObjectEncoder) AddInt16(key string, value int16) {
	_ = o.EncodeKeyval(key, value)
}

func (o ObjectEncoder) AddInt8(key string, value int8) {
	_ = o.EncodeKeyval(key, value)
}

func (o *ObjectEncoder) AddString(key, value string) {
	if !strings.Contains(value, "\n") {
		_ = o.EncodeKeyval(key, value)
		return
	}

	split := strings.Split(value, "\n")
	if len(split) == 1 {
		_ = o.EncodeKeyval(key, split[0])
		return
	}

	_ = o.EndRecord()
	_ = o.EncodeKeyval(key, "")
	o.buf.WriteString(value)
	_ = o.EndRecord()
}

func (o ObjectEncoder) AddTime(key string, value time.Time) {
	_ = o.EncodeKeyval(key, value.Format(time.RFC3339))
}

func (o ObjectEncoder) AddUint(key string, value uint) {
	_ = o.EncodeKeyval(key, value)
}

func (o ObjectEncoder) AddUint64(key string, value uint64) {
	_ = o.EncodeKeyval(key, value)
}

func (o ObjectEncoder) AddUint32(key string, value uint32) {
	_ = o.EncodeKeyval(key, value)
}

func (o ObjectEncoder) AddUint16(key string, value uint16) {
	_ = o.EncodeKeyval(key, value)
}

func (o ObjectEncoder) AddUint8(key string, value uint8) {
	_ = o.EncodeKeyval(key, value)
}

func (o ObjectEncoder) AddUintptr(key string, value uintptr) {
	_ = o.EncodeKeyval(key, uint64(value))
}

func (o *ObjectEncoder) AddReflected(key string, value interface{}) error {
	o.resetReflectBuf()

	if err := o.reflectEnc.Encode(value); err != nil {
		return errors.WithStack(err)
	}

	o.reflectBuf.TrimNewline()

	return o.EncodeKeyval(key, string(o.reflectBuf.Bytes()))
}

func (o ObjectEncoder) OpenNamespace(key string) {
	_ = o.EndRecord()
}
