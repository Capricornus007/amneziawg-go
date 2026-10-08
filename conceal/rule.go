package conceal

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var (
	errInvalidData = errors.New("invalid data")
)

type readContext struct {
	FlexBuffer
	*BufferPool
	nextDataSize int
	formatData   []byte
}

func (ctx *readContext) rememberRead(b []byte) {
	ctx.formatData = append(ctx.formatData, b...)
}

type writeContext struct {
	FlexBuffer
	*BufferPool

	// Counter backs the AWG <c> tag. The writer seeds it with a random value
	// once per decoy group and bumps it after every decoy, which reproduces the
	// packet counter the device-side implementation used to embed.
	Counter uint32
}

type Rule interface {
	Spec() string
	Write(w io.Writer, ctx *writeContext) error
	Read(r io.Reader, ctx *readContext) error
}

type Rules []Rule

func (r Rules) Spec() string {
	var builder strings.Builder
	for _, rule := range r {
		builder.WriteString(rule.Spec())
	}
	return builder.String()
}

func (r Rules) Write(w io.Writer, ctx *writeContext) error {
	for _, rule := range r {
		if err := rule.Write(w, ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r Rules) Read(rd io.Reader, ctx *readContext) error {
	formatDataStart := len(ctx.formatData)
	for _, rule := range r {
		if err := rule.Read(rd, ctx); err != nil {
			if errors.Is(err, ErrFormat) {
				return err
			}
			if errors.Is(err, errInvalidData) {
				return NewFormatError(ctx.formatData[formatDataStart:], err)
			}
			return err
		}
	}
	return nil
}

func (r Rules) Match(b []byte, pool *BufferPool) bool {
	if r == nil {
		return false
	}
	tmp := pool.Get()
	defer pool.Put(tmp)

	reader := newSliceReader(b)
	ctx := readContext{
		FlexBuffer: WrapFlexBuffer(tmp),
		BufferPool: pool,
	}
	if err := r.Read(&reader, &ctx); err != nil {
		return false
	}
	return len(reader.buf) == 0
}

func buildBytesRule(val string) (Rule, error) {
	val = strings.TrimPrefix(val, "0x")

	if len(val) == 0 {
		return nil, errors.New("empty argument")
	}

	if len(val)%2 != 0 {
		return nil, errors.New("odd amount of symbols")
	}

	bytes, err := hex.DecodeString(val)
	if err != nil {
		return nil, err
	}

	return &bytesRule{data: bytes}, nil
}

type bytesRule struct {
	data []byte
}

func (r *bytesRule) Spec() string {
	return fmt.Sprintf("<b 0x%x>", r.data)
}

func (r *bytesRule) Write(w io.Writer, ctx *writeContext) error {
	_, err := w.Write(r.data)
	return err
}

func (r *bytesRule) Read(rd io.Reader, ctx *readContext) error {
	tmp := ctx.Get()
	defer ctx.Put(tmp)

	buf := tmp[:len(r.data)]
	n, err := io.ReadFull(rd, buf)
	ctx.rememberRead(buf[:n])
	if err != nil {
		return err
	}

	if !bytes.Equal(buf, r.data) {
		return errInvalidData
	}

	return nil
}

func buildRandRule(val string) (Rule, error) {
	length, err := strconv.Atoi(val)
	if err != nil {
		return nil, err
	}

	return &randRule{length: length}, nil
}

type randRule struct {
	length int
}

func (r *randRule) Spec() string {
	return fmt.Sprintf("<r %d>", r.length)
}

func (r *randRule) Write(w io.Writer, ctx *writeContext) error {
	tmp := ctx.Get()
	defer ctx.Put(tmp)

	buf := tmp[:r.length]
	rand.Read(buf)

	_, err := w.Write(buf)
	return err
}

func (r *randRule) Read(rd io.Reader, ctx *readContext) error {
	tmp := ctx.Get()
	defer ctx.Put(tmp)

	buf := tmp[:r.length]
	n, err := io.ReadFull(rd, buf)
	ctx.rememberRead(buf[:n])
	if err != nil {
		return err
	}

	// I guess, there is no way to validate randomness
	// so just return nil here like everything is fine
	return nil
}

func buildRandDigitsRule(val string) (Rule, error) {
	length, err := strconv.Atoi(val)
	if err != nil {
		return nil, err
	}

	return &randDigitRule{length: length}, nil
}

type randDigitRule struct {
	length int
}

func (r *randDigitRule) Spec() string {
	return fmt.Sprintf("<rd %d>", r.length)
}

const digits10 = "0123456789"

func (r *randDigitRule) Write(w io.Writer, ctx *writeContext) error {
	tmp := ctx.Get()
	defer ctx.Put(tmp)

	buf := tmp[:r.length]
	rand.Read(buf)
	for i := range buf {
		buf[i] = digits10[buf[i]%10]
	}

	_, err := w.Write(buf)
	return err
}

func (r *randDigitRule) Read(rd io.Reader, ctx *readContext) error {
	tmp := ctx.Get()
	defer ctx.Put(tmp)

	buf := tmp[:r.length]
	n, err := io.ReadFull(rd, buf)
	ctx.rememberRead(buf[:n])
	if err != nil {
		return err
	}

	for _, b := range buf {
		if !unicode.IsDigit(rune(b)) {
			return errInvalidData
		}
	}

	return nil
}

func buildRandCharRule(val string) (Rule, error) {
	length, err := strconv.Atoi(val)
	if err != nil {
		return nil, err
	}

	return &randCharRule{length: length}, nil
}

type randCharRule struct {
	length int
}

func (r *randCharRule) Spec() string {
	return fmt.Sprintf("<rc %d>", r.length)
}

const chars52 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func (r *randCharRule) Write(w io.Writer, ctx *writeContext) error {
	tmp := ctx.Get()
	defer ctx.Put(tmp)

	buf := tmp[:r.length]
	rand.Read(buf)
	for i := range buf {
		buf[i] = chars52[buf[i]%52]
	}

	_, err := w.Write(buf)
	return err
}

func (r *randCharRule) Read(rd io.Reader, ctx *readContext) error {
	tmp := ctx.Get()
	defer ctx.Put(tmp)

	buf := tmp[:r.length]
	n, err := io.ReadFull(rd, buf)
	ctx.rememberRead(buf[:n])
	if err != nil {
		return err
	}

	for _, b := range buf {
		if !unicode.IsLetter(rune(b)) {
			return errInvalidData
		}
	}

	return nil
}

func buildTimestampRule(_ string) (Rule, error) {
	return &timestampRule{}, nil
}

type timestampRule struct{}

func (r *timestampRule) Spec() string {
	return "<t>"
}

func (r *timestampRule) Write(w io.Writer, ctx *writeContext) error {
	timestamp := uint32(time.Now().Unix())
	return binary.Write(w, binary.BigEndian, timestamp)
}

func (r *timestampRule) Read(rd io.Reader, ctx *readContext) error {
	tmp := ctx.Get()
	defer ctx.Put(tmp)

	buf := tmp[:4]
	n, err := io.ReadFull(rd, buf)
	ctx.rememberRead(buf[:n])
	if err != nil {
		return err
	}
	_ = binary.BigEndian.Uint32(buf)

	// TODO: check timestamp?

	return nil
}

func buildDataSizeRule(val string) (Rule, error) {
	var (
		length int       = 2
		format NumFormat = NumFormatBE
		end    byte      = 0
		err    error
	)

	parts := strings.Fields(val)
	switch len(parts) {
	case 1:
		// Legacy AWG spelling: "<dz 4>" means a 4-byte big-endian size field,
		// which the newer "<dz be 4>" form says explicitly. Kept working so old
		// configs do not silently lose their framing.
		if length, err = strconv.Atoi(parts[0]); err != nil {
			return nil, err
		}
		return &dataSizeRule{
			length: length,
			format: NumFormatBE,
			end:    end,
		}, nil
	case 2:
	default:
		return nil, errors.New("wrong amount of arguments")
	}

	if format, err = buildNumFormat(parts[0]); err != nil {
		return nil, err
	}

	switch format {
	case NumFormatAscii, NumFormatHex:
		parts[1] = strings.TrimPrefix(parts[1], "0x")

		var bytes []byte
		bytes, err = hex.DecodeString(parts[1])
		if err != nil {
			return nil, err
		}

		if len(bytes) != 1 {
			return nil, errors.New("too many bytes")
		}

		end = bytes[0]
	default:
		if length, err = strconv.Atoi(parts[1]); err != nil {
			return nil, err
		}
	}

	return &dataSizeRule{
		length: length,
		format: format,
		end:    end,
	}, nil
}

type dataSizeRule struct {
	format NumFormat
	length int
	end    byte
}

func (r *dataSizeRule) Spec() string {
	switch r.format {
	case NumFormatAscii, NumFormatHex:
		return fmt.Sprintf("<dz %s 0x%02x>", r.format.Spec(), r.end)
	}
	return fmt.Sprintf("<dz %s %d>", r.format.Spec(), r.length)
}

func (r *dataSizeRule) Write(w io.Writer, ctx *writeContext) error {
	tmp := ctx.Get()
	defer ctx.Put(tmp)

	size := int32(ctx.Cap())

	switch r.format {
	case NumFormatBE:
		for i := r.length - 1; i >= 0; i-- {
			tmp[i] = byte(size & 0xFF)
			size >>= 8
		}
		if _, err := w.Write(tmp[:r.length]); err != nil {
			return err
		}
	case NumFormatLE:
		for i := range r.length {
			tmp[i] = byte(size & 0xFF)
			size >>= 8
		}
		if _, err := w.Write(tmp[:r.length]); err != nil {
			return err
		}
	case NumFormatAscii:
		b := strconv.AppendInt(tmp[:0], int64(size), 10)
		b = append(b, r.end)

		if _, err := w.Write(b); err != nil {
			return err
		}
	case NumFormatHex:
		b := strconv.AppendInt(tmp[:0], int64(size), 16)
		b = append(b, r.end)

		if _, err := w.Write(b); err != nil {
			return err
		}
	}

	return nil
}

func (r *dataSizeRule) Read(rd io.Reader, ctx *readContext) error {
	tmp := ctx.Get()
	defer ctx.Put(tmp)

	switch r.format {
	case NumFormatBE:
		buf := tmp[:r.length]
		n, err := io.ReadFull(rd, buf)
		ctx.rememberRead(buf[:n])
		if err != nil {
			if errors.Is(err, io.ErrShortBuffer) {
				return errInvalidData
			}
			return err
		}
		var size int
		for i := range buf {
			size <<= 8
			size |= int(buf[i])
		}
		ctx.nextDataSize = size

	case NumFormatLE:
		buf := tmp[:r.length]
		n, err := io.ReadFull(rd, buf)
		ctx.rememberRead(buf[:n])
		if err != nil {
			if errors.Is(err, io.ErrShortBuffer) {
				return errInvalidData
			}
			return err
		}
		var size int
		for i := len(buf) - 1; i >= 0; i-- {
			size <<= 8
			size |= int(buf[i])
		}
		ctx.nextDataSize = size

	case NumFormatAscii:
		n, err := ReadUntil(rd, tmp, r.end)
		if err == nil {
			ctx.rememberRead(tmp[:n+1])
		} else {
			ctx.rememberRead(tmp[:n])
		}
		if err != nil {
			return err
		}

		size64, err := strconv.ParseInt(string(tmp[:n]), 10, 32)
		if err != nil {
			return fmt.Errorf("%w: %v", errInvalidData, err)
		}
		ctx.nextDataSize = int(size64)

	case NumFormatHex:
		n, err := ReadUntil(rd, tmp, r.end)
		if err == nil {
			ctx.rememberRead(tmp[:n+1])
		} else {
			ctx.rememberRead(tmp[:n])
		}
		if err != nil {
			return err
		}

		size64, err := strconv.ParseInt(string(tmp[:n]), 16, 32)
		if err != nil {
			return fmt.Errorf("%w: %v", errInvalidData, err)
		}
		ctx.nextDataSize = int(size64)
	}

	return nil
}

func buildDataRule(val string) (Rule, error) {
	return &dataRule{}, nil
}

type dataRule struct{}

func (r *dataRule) Spec() string {
	return "<d>"
}

func (r *dataRule) Write(w io.Writer, ctx *writeContext) error {
	buf := ctx.PullHead(-1)
	if buf == nil {
		return io.ErrShortBuffer
	}

	_, err := w.Write(buf)
	return err
}

func (r *dataRule) Read(rd io.Reader, ctx *readContext) error {
	buf := ctx.PushTail(ctx.nextDataSize)
	if buf == nil {
		return errInvalidData
	}

	n, err := io.ReadFull(rd, buf)
	ctx.rememberRead(buf[:n])
	return err
}

// buildCounterRule implements the AWG <c> tag: a 4-byte big-endian packet
// counter. The value comes from writeContext.Counter, which the decoy writer
// seeds randomly per message and increments per decoy packet, matching the
// device-side AWG 3.x behaviour this rule was ported from.
func buildCounterRule(_ string) (Rule, error) {
	return &counterRule{}, nil
}

type counterRule struct{}

func (r *counterRule) Spec() string {
	return "<c>"
}

func (r *counterRule) Write(w io.Writer, ctx *writeContext) error {
	tmp := ctx.Get()
	defer ctx.Put(tmp)

	binary.BigEndian.PutUint32(tmp[:4], ctx.Counter)
	_, err := w.Write(tmp[:4])
	return err
}

func (r *counterRule) Read(rd io.Reader, ctx *readContext) error {
	tmp := ctx.Get()
	defer ctx.Put(tmp)

	buf := tmp[:4]
	n, err := io.ReadFull(rd, buf)
	ctx.rememberRead(buf[:n])
	if err != nil {
		return err
	}

	// The counter carries no payload; it is only checked for presence, exactly
	// like the device-side obfuscator it replaces.
	return nil
}

// buildDataStringRule implements the AWG <ds> tag: the payload written as an
// unpadded base64 string. It pairs with <dz>, which states the raw size.
func buildDataStringRule(val string) (Rule, error) {
	return &dataStringRule{}, nil
}

type dataStringRule struct{}

func (r *dataStringRule) Spec() string {
	return "<ds>"
}

func (r *dataStringRule) Write(w io.Writer, ctx *writeContext) error {
	buf := ctx.PullHead(-1)
	if buf == nil {
		return io.ErrShortBuffer
	}

	tmp := ctx.Get()
	defer ctx.Put(tmp)

	encoded := base64.RawStdEncoding.EncodeToString(buf)
	if len(encoded) > len(tmp) {
		return io.ErrShortBuffer
	}

	_, err := w.Write([]byte(encoded))
	return err
}

func (r *dataStringRule) Read(rd io.Reader, ctx *readContext) error {
	if ctx.nextDataSize <= 0 {
		return errInvalidData
	}

	raw := ctx.PushTail(ctx.nextDataSize)
	if raw == nil {
		return errInvalidData
	}

	tmp := ctx.Get()
	defer ctx.Put(tmp)

	encodedLen := base64.RawStdEncoding.EncodedLen(ctx.nextDataSize)
	if encodedLen > len(tmp) {
		return errInvalidData
	}

	buf := tmp[:encodedLen]
	n, err := io.ReadFull(rd, buf)
	ctx.rememberRead(buf[:n])
	if err != nil {
		return err
	}

	if _, err := base64.RawStdEncoding.Decode(raw, buf); err != nil {
		return fmt.Errorf("%w: %v", errInvalidData, err)
	}

	return nil
}

// seedDecoyCounter gives one decoy group a fresh random <c> base value, the way
// the device-side AWG implementation did for every handshake message. The
// emitter then bumps the counter once per decoy packet it writes.
func seedDecoyCounter(ctx *writeContext) {
	var seed [4]byte
	rand.Read(seed[:])
	ctx.Counter = binary.BigEndian.Uint32(seed[:])
}
