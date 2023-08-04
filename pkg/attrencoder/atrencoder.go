package attrencoder

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

const (
	CallerKey     = "_caller"
	StacktraceKey = "stack"
	MsgKey        = "msg"
)

const (
	placeholderInvalidUTF8 = "<invalid_utf8>"
)

var (
	_pool = buffer.NewPool()
	// Get retrieves a buffer from the pool, creating one if necessary.
	Get = _pool.Get

	_ zapcore.ObjectEncoder = &AtrEncoder{}
)

type AtrEncoder struct {
	attrs []attribute.KeyValue

	// for encoding generic values by reflection
	reflectBuf *buffer.Buffer
	reflectEnc *json.Encoder
}

func NewAttr(attr ...attribute.KeyValue) *AtrEncoder {
	return &AtrEncoder{attrs: attr}
}

func (a *AtrEncoder) Clone() *AtrEncoder {
	v := make([]attribute.KeyValue, len(a.attrs))
	copy(v, a.attrs)

	n := NewAttr(v...)

	return n
}

func (a *AtrEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) ([]attribute.KeyValue, error) {
	n := a.Clone()

	if ent.Caller.Defined {
		zap.String(CallerKey, ent.Caller.TrimmedPath()).AddTo(n)
	}

	if len(ent.Stack) > 0 {
		zap.String(StacktraceKey, ent.Stack).AddTo(n)
	}

	zap.String(MsgKey, ent.Message).AddTo(n)

	for _, field := range fields {
		field.AddTo(n)
	}

	return n.attrs, nil
}

func (a *AtrEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error {
	a.attrs = append(a.attrs, attribute.String(key, fmt.Sprintf("%v", arr)))
	return nil
}

func (a *AtrEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	a.attrs = append(a.attrs, attribute.String(key, fmt.Sprintf("%v", obj)))
	return nil
}

func (a *AtrEncoder) AddBinary(key string, value []byte) {
	a.attrs = append(a.attrs, attribute.String(key, base64.StdEncoding.EncodeToString(value)))
}

func (a *AtrEncoder) AddByteString(key string, value []byte) {
	if !utf8.Valid(value) {
		a.attrs = append(a.attrs, attribute.String(key, placeholderInvalidUTF8))
		return
	}

	a.attrs = append(a.attrs, attribute.String(key, string(value)))
}

func (a *AtrEncoder) AddBool(key string, value bool) {
	a.attrs = append(a.attrs, attribute.Bool(key, value))
}

func (a *AtrEncoder) AddComplex128(key string, value complex128) {
	a.attrs = append(a.attrs, attribute.String(key, strconv.FormatComplex(value, 'g', -1, 128)))
}

func (a *AtrEncoder) AddComplex64(key string, value complex64) {
	a.attrs = append(a.attrs, attribute.String(key,
		strconv.FormatComplex(complex128(value), 'g', -1, 64)))
}

func (a *AtrEncoder) AddDuration(key string, value time.Duration) {
	a.attrs = append(a.attrs, attribute.String(key, value.String()))
}

func (a *AtrEncoder) AddFloat64(key string, value float64) {
	a.attrs = append(a.attrs, attribute.Float64(key, value))
}

func (a *AtrEncoder) AddFloat32(key string, value float32) {
	a.attrs = append(a.attrs, attribute.Float64(key, float64(value)))
}

func (a *AtrEncoder) AddInt(key string, value int) {
	a.attrs = append(a.attrs, attribute.Int(key, value))
}

func (a *AtrEncoder) AddInt64(key string, value int64) {
	a.attrs = append(a.attrs, attribute.Int64(key, value))
}

func (a *AtrEncoder) AddInt32(key string, value int32) {
	a.attrs = append(a.attrs, attribute.Int(key, int(value)))
}

func (a *AtrEncoder) AddInt16(key string, value int16) {
	a.attrs = append(a.attrs, attribute.Int(key, int(value)))
}

func (a *AtrEncoder) AddInt8(key string, value int8) {
	a.attrs = append(a.attrs, attribute.Int(key, int(value)))
}

// AddString has multi-line issue
func (a *AtrEncoder) AddString(key, value string) {
	//if strings.Contains(value, " ") {
	//	a.attrs = append(a.attrs, attribute.String(key, fmt.Sprintf(`"%s"`, value)))
	//	return
	//}

	if !utf8.ValidString(value) {
		a.attrs = append(a.attrs, attribute.String(key, placeholderInvalidUTF8))
		return
	}

	a.attrs = append(a.attrs, attribute.String(key, value))
}

func (a *AtrEncoder) AddTime(key string, value time.Time) {
	a.attrs = append(a.attrs, attribute.String(key, value.Format(time.RFC3339)))
}

func (a *AtrEncoder) AddUint(key string, value uint) {
	a.attrs = append(a.attrs, attribute.Int(key, int(value)))
}

func (a *AtrEncoder) AddUint64(key string, value uint64) {
	a.attrs = append(a.attrs, attribute.Int(key, int(value)))
}

func (a *AtrEncoder) AddUint32(key string, value uint32) {
	a.attrs = append(a.attrs, attribute.Int(key, int(value)))
}

func (a *AtrEncoder) AddUint16(key string, value uint16) {
	a.attrs = append(a.attrs, attribute.Int(key, int(value)))
}

func (a *AtrEncoder) AddUint8(key string, value uint8) {
	a.attrs = append(a.attrs, attribute.Int(key, int(value)))
}

func (a *AtrEncoder) AddUintptr(key string, value uintptr) {
	a.attrs = append(a.attrs, attribute.Int(key, int(value)))
}

func (a *AtrEncoder) AddReflected(key string, value interface{}) error {
	a.resetReflectBuf()

	if err := a.reflectEnc.Encode(value); err != nil {
		return errors.WithStack(err)
	}

	a.reflectBuf.TrimNewline()
	a.attrs = append(a.attrs, attribute.String(key, a.reflectBuf.String()))

	return nil
}

// OpenNamespace not used, for json flow only
func (a *AtrEncoder) OpenNamespace(key string) {}

func (a *AtrEncoder) resetReflectBuf() {
	if a.reflectBuf == nil {
		a.reflectBuf = Get()
		a.reflectEnc = json.NewEncoder(a.reflectBuf)

		// For consistency with our custom JSON encoder.
		a.reflectEnc.SetEscapeHTML(false)
	} else {
		a.reflectBuf.Reset()
	}
}
