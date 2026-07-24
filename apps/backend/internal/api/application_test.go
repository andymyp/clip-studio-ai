package api

import (
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

type stringEncoder struct {
	value string
}

func (encoder *stringEncoder) AppendString(value string) {
	encoder.value = value
}

func TestEncodeLogTime(t *testing.T) {
	value := time.Date(2026, time.July, 24, 14, 48, 31, 0, time.Local)
	encoder := &stringEncoder{}

	encodeLogTime(value, encoder)

	if encoder.value != "2026-07-24 14:48" {
		t.Fatalf("encoded time = %q", encoder.value)
	}
}

var _ zapcore.PrimitiveArrayEncoder = (*stringEncoder)(nil)

func (encoder *stringEncoder) AppendBool(bool)              {}
func (encoder *stringEncoder) AppendByteString([]byte)      {}
func (encoder *stringEncoder) AppendComplex128(complex128)  {}
func (encoder *stringEncoder) AppendComplex64(complex64)    {}
func (encoder *stringEncoder) AppendFloat64(float64)        {}
func (encoder *stringEncoder) AppendFloat32(float32)        {}
func (encoder *stringEncoder) AppendInt(int)                {}
func (encoder *stringEncoder) AppendInt64(int64)            {}
func (encoder *stringEncoder) AppendInt32(int32)            {}
func (encoder *stringEncoder) AppendInt16(int16)            {}
func (encoder *stringEncoder) AppendInt8(int8)              {}
func (encoder *stringEncoder) AppendUint(uint)              {}
func (encoder *stringEncoder) AppendUint64(uint64)          {}
func (encoder *stringEncoder) AppendUint32(uint32)          {}
func (encoder *stringEncoder) AppendUint16(uint16)          {}
func (encoder *stringEncoder) AppendUint8(uint8)            {}
func (encoder *stringEncoder) AppendUintptr(uintptr)        {}
func (encoder *stringEncoder) AppendDuration(time.Duration) {}
func (encoder *stringEncoder) AppendTime(time.Time)         {}
func (encoder *stringEncoder) AppendReflected(any) error    { return nil }
func (encoder *stringEncoder) AppendArray(zapcore.ArrayMarshaler) error {
	return nil
}
