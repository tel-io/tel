package tel

import (
	"go.uber.org/zap"
)

/*
	Create internal zap.Fields functions
	It's just helpers which allow not use zap packet at all inside project which uses tel

	As we planning to use otel attribute.KeyValue in future it could be replaces
*/

var (
	Any        = zap.Any
	Binary     = zap.Binary
	ByteString = zap.ByteString
	Bool       = zap.Bool
	Duration   = zap.Duration
	Float32    = zap.Float32
	Float64    = zap.Float64
	Int        = zap.Int
	Int64      = zap.Int64
	Int32      = zap.Int32
	Int16      = zap.Int16
	Int8       = zap.Int8
	String     = zap.String
	Time       = zap.Time
	Uint       = zap.Uint
	Uint64     = zap.Uint64
	Uint32     = zap.Uint32
	Uint16     = zap.Uint16
	Uint8      = zap.Uint8
	Uintptr    = zap.Uintptr
	Error      = zap.Error
)

// Arrays implementation

var (
	Strings = zap.Strings
	Ints    = zap.Ints
)
