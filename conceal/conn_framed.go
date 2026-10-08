package conceal

import (
	"crypto/rand"
	"encoding/binary"
	"net"
	"sync"

	"golang.org/x/crypto/chacha20"
	"golang.org/x/net/ipv4"
)

// AWG 3.1 header protection (formerly applied inside the device layer).
// The keystream is ChaCha20 keyed by the header protection key, with the nonce
// taken from the first HeaderProtectionNonceSize bytes of the per-message
// padding prefix that the frame encoder generates.
const (
	HeaderProtectionKeySize   = 32
	HeaderProtectionNonceSize = 12
)

type FramedOpts struct {
	H1           *RangedHeader
	H2           *RangedHeader
	H3           *RangedHeader
	H4           *RangedHeader
	S1           int
	S2           int
	S3           int
	S4           int
	HeaderCompat bool

	// AWG 3.1 extras carried over from the device-side implementation.
	HeaderProtectionKey [HeaderProtectionKeySize]byte
	HasHeaderProtection bool
	RandomTrailers      bool
}

func (o *FramedOpts) HasIntersections() bool {
	headers := []*RangedHeader{o.H1, o.H2, o.H3, o.H4}

	for i := range len(headers) {
		left := headers[i]
		if left == nil {
			continue
		}

		for j := i + 1; j < len(headers); j++ {
			right := headers[j]
			if right == nil {
				continue
			}

			if left.start <= right.end && right.start <= left.end {
				return true
			}
		}
	}

	return false
}

func newFrameEncoding(opts FramedOpts) (e frameEncoding, ok bool) {
	e = frameEncoding{}

	if opts.H1 != nil {
		e.header.initial = *opts.H1
		ok = true
	} else {
		e.header.initial = RangedHeader{WireguardMsgInitiationType, WireguardMsgInitiationType}
	}

	if opts.H2 != nil {
		e.header.response = *opts.H2
		ok = true
	} else {
		e.header.response = RangedHeader{WireguardMsgResponseType, WireguardMsgResponseType}
	}

	if opts.H3 != nil {
		e.header.cookie = *opts.H3
		ok = true
	} else {
		e.header.cookie = RangedHeader{WireguardMsgCookieReplyType, WireguardMsgCookieReplyType}
	}

	if opts.H4 != nil {
		e.header.transport = *opts.H4
		ok = true
	} else {
		e.header.transport = RangedHeader{WireguardMsgTransportType, WireguardMsgTransportType}
	}

	if opts.S1 != 0 {
		e.padding.initial = opts.S1
		ok = true
	}

	if opts.S2 != 0 {
		e.padding.response = opts.S2
		ok = true
	}

	if opts.S3 != 0 {
		e.padding.cookie = opts.S3
		ok = true
	}

	if opts.S4 != 0 {
		e.padding.transport = opts.S4
		ok = true
	}

	// Header protection needs a padding prefix to derive its nonce from, so it
	// only makes sense together with S1-S4. Enabling it still turns the framing
	// stage on even when no header/padding was configured, so the config is
	// never silently ignored.
	if opts.HasHeaderProtection && !isZeroKey(opts.HeaderProtectionKey) {
		e.hpKey = opts.HeaderProtectionKey
		e.hasHP = true
		ok = true
	}

	if opts.RandomTrailers {
		e.trailers = true
	}

	e.compat = opts.HeaderCompat
	return e, ok
}

func isZeroKey(key [HeaderProtectionKeySize]byte) bool {
	var zero [HeaderProtectionKeySize]byte
	return key == zero
}

type frameEncoding struct {
	header struct {
		initial   RangedHeader
		response  RangedHeader
		cookie    RangedHeader
		transport RangedHeader
	}
	padding struct {
		initial   int
		response  int
		cookie    int
		transport int
	}
	compat   bool
	hpKey    [HeaderProtectionKeySize]byte
	hasHP    bool
	trailers bool
}

type frameRecordKind uint8

const (
	frameRecordInvalid frameRecordKind = iota
	frameRecordInitiation
	frameRecordResponse
	frameRecordCookie
	frameRecordTransport
)

// messageSize returns the on-wire size of the WireGuard message the given kind
// carries, excluding padding and random trailers.
func (k frameRecordKind) messageSize() int {
	switch k {
	case frameRecordInitiation:
		return WireguardMsgInitiationSize
	case frameRecordResponse:
		return WireguardMsgResponseSize
	case frameRecordCookie:
		return WireguardMsgCookieReplySize
	case frameRecordTransport:
		return WireguardMsgTransportHeaderSize
	}
	return 0
}

func paddingOf(e *frameEncoding, kind frameRecordKind) int {
	switch kind {
	case frameRecordInitiation:
		return e.padding.initial
	case frameRecordResponse:
		return e.padding.response
	case frameRecordCookie:
		return e.padding.cookie
	case frameRecordTransport:
		return e.padding.transport
	}
	return 0
}

func headerOf(e *frameEncoding, kind frameRecordKind) RangedHeader {
	switch kind {
	case frameRecordInitiation:
		return e.header.initial
	case frameRecordResponse:
		return e.header.response
	case frameRecordCookie:
		return e.header.cookie
	case frameRecordTransport:
		return e.header.transport
	}
	return RangedHeader{}
}

// applyHeaderProtection runs the AWG 3.1 keystream over the covered part of the
// message body. body must start right after the padding prefix (i.e. at the
// message type field) and nonce must be the padding prefix itself.
func (e *frameEncoding) applyHeaderProtection(nonce []byte, body []byte, kind frameRecordKind) bool {
	if !e.hasHP {
		return true
	}
	if len(nonce) < HeaderProtectionNonceSize || len(body) < kind.messageSize() {
		return false
	}
	cip, err := chacha20.NewUnauthenticatedCipher(e.hpKey[:], nonce[:HeaderProtectionNonceSize])
	if err != nil {
		return false
	}
	cip.XORKeyStream(body[:kind.messageSize()], body[:kind.messageSize()])
	return true
}

// encodeOne builds one framed record. When replaceHeader is set the message type
// field is swapped for a value generated from the configured header range
// (non-compat framing); otherwise the source is copied as-is (compat framing,
// where the device already picked the header value). A random trailer appended
// by the device travels untouched through the keystream-covered region.
func (e *frameEncoding) encodeOne(dst, src []byte, kind frameRecordKind, replaceHeader bool) int {
	padding := paddingOf(e, kind)
	if len(dst) < padding+len(src) || len(src) < 4 {
		return 0
	}

	rand.Read(dst[:padding])

	body := dst[padding:]
	var n int
	if replaceHeader {
		header := headerOf(e, kind)
		binary.LittleEndian.PutUint32(body[:4], header.Generate())
		n = 4 + copy(body[4:], src[4:])
	} else {
		n = copy(body, src)
	}

	if !e.applyHeaderProtection(dst[:padding], body[:n], kind) {
		return 0
	}
	return padding + n
}

func (e *frameEncoding) Encode(dst, src []byte) int {
	if len(src) < 4 {
		return 0
	}

	header := binary.LittleEndian.Uint32(src[:4])

	if e.compat {
		switch {
		case e.header.initial.Validate(header):
			return e.encodeOne(dst, src, frameRecordInitiation, false)
		case e.header.response.Validate(header):
			return e.encodeOne(dst, src, frameRecordResponse, false)
		case e.header.cookie.Validate(header):
			return e.encodeOne(dst, src, frameRecordCookie, false)
		case e.header.transport.Validate(header):
			return e.encodeOne(dst, src, frameRecordTransport, false)
		}
	} else {
		switch src[0] {
		case WireguardMsgInitiationType:
			return e.encodeOne(dst, src, frameRecordInitiation, true)
		case WireguardMsgResponseType:
			return e.encodeOne(dst, src, frameRecordResponse, true)
		case WireguardMsgCookieReplyType:
			return e.encodeOne(dst, src, frameRecordCookie, true)
		case WireguardMsgTransportType:
			return e.encodeOne(dst, src, frameRecordTransport, true)
		}
	}

	return 0
}

// validateHeader checks the (possibly header-protected) message type field
// without touching the packet.
func (e *frameEncoding) validateHeader(nonce []byte, typeField []byte, header RangedHeader) bool {
	if len(typeField) < 4 {
		return false
	}
	if !e.hasHP {
		return header.Validate(binary.LittleEndian.Uint32(typeField))
	}
	if len(nonce) < HeaderProtectionNonceSize {
		return false
	}
	cip, err := chacha20.NewUnauthenticatedCipher(e.hpKey[:], nonce[:HeaderProtectionNonceSize])
	if err != nil {
		return false
	}
	var plain [4]byte
	cip.XORKeyStream(plain[:], typeField[:4])
	return header.Validate(binary.LittleEndian.Uint32(plain[:]))
}

// decodeOne strips padding and the random trailer, running the header
// protection keystream backwards on the way.
func (e *frameEncoding) decodeOne(b []byte, kind frameRecordKind, originalHeader uint32) int {
	padding := paddingOf(e, kind)
	messageSize := kind.messageSize()
	if len(b) < padding+4 {
		return 0
	}

	body := b[padding:]
	if kind != frameRecordTransport && len(body) > messageSize {
		if !e.trailers {
			return 0
		}
		body = body[:messageSize]
	}

	if !e.applyHeaderProtection(b[:padding], body, kind) {
		return 0
	}

	if e.compat {
		return copy(b, body)
	}

	binary.LittleEndian.PutUint32(b[:4], originalHeader)
	n := copy(b[4:], body[4:])

	return 4 + n
}

func (e *frameEncoding) recordSize(kind frameRecordKind) int {
	return paddingOf(e, kind) + kind.messageSize()
}

func (e *frameEncoding) matchesRecord(b []byte, kind frameRecordKind) bool {
	size := e.recordSize(kind)
	padding := paddingOf(e, kind)

	if len(b) != size && !(e.trailers && len(b) > size) {
		return false
	}
	if len(b) < padding+4 {
		return false
	}
	return e.validateHeader(b[:padding], b[padding:padding+4], headerOf(e, kind))
}

func (e *frameEncoding) matchesTransportRecord(b []byte) bool {
	padding := e.padding.transport
	if len(b) < WireguardMsgTransportMinSize+padding {
		return false
	}
	if len(b) < padding+4 {
		return false
	}
	return e.validateHeader(b[:padding], b[padding:padding+4], e.header.transport)
}

func (e *frameEncoding) recordKind(b []byte) frameRecordKind {
	if e.matchesRecord(b, frameRecordInitiation) {
		return frameRecordInitiation
	}
	if e.matchesRecord(b, frameRecordResponse) {
		return frameRecordResponse
	}
	if e.matchesRecord(b, frameRecordCookie) {
		return frameRecordCookie
	}
	if e.matchesTransportRecord(b) {
		return frameRecordTransport
	}
	return frameRecordInvalid
}

func (e *frameEncoding) IsValidRecord(b []byte) bool {
	return e.recordKind(b) != frameRecordInvalid
}

func (e *frameEncoding) IsInitiationRecord(b []byte) bool {
	return e.recordKind(b) == frameRecordInitiation
}

func (e *frameEncoding) Decode(b []byte) (int, error) {
	switch kind := e.recordKind(b); kind {
	case frameRecordInitiation:
		return e.decodeOne(b, kind, WireguardMsgInitiationType), nil
	case frameRecordResponse:
		return e.decodeOne(b, kind, WireguardMsgResponseType), nil
	case frameRecordCookie:
		return e.decodeOne(b, kind, WireguardMsgCookieReplyType), nil
	case frameRecordTransport:
		return e.decodeOne(b, kind, WireguardMsgTransportType), nil
	default:
		return 0, NewFormatError(b, errInvalidData)
	}
}

func NewFramedConn(conn net.Conn, pool *sync.Pool, opts FramedOpts) (c *FramedConn, ok bool) {
	enc, ok := newFrameEncoding(opts)
	if !ok {
		return nil, false
	}

	return &FramedConn{
		Conn: conn,
		pool: WrapBufferPool(pool),
		enc:  enc,
	}, true
}

type FramedConn struct {
	net.Conn
	pool *BufferPool
	enc  frameEncoding
}

func (c *FramedConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if n > 0 {
		var decodeErr error
		n, decodeErr = c.enc.Decode(b[:n])
		if decodeErr != nil {
			return 0, decodeErr
		}
	}
	return n, err
}

func (c *FramedConn) Write(b []byte) (n int, err error) {
	t := c.pool.Get()
	defer c.pool.Put(t)

	n = c.enc.Encode(t, b)
	diff := n - len(b)
	n, err = c.Conn.Write(t[:n])

	return max(n-diff, 0), err
}

func NewFramedUDPConn(conn UDPConn, pool *sync.Pool, opts FramedOpts) (c UDPConn, ok bool) {
	enc, ok := newFrameEncoding(opts)
	if !ok {
		return nil, false
	}

	return &FramedUDPConn{
		UDPConn: conn,
		pool:    WrapBufferPool(pool),
		enc:     enc,
	}, true
}

type FramedUDPConn struct {
	UDPConn
	pool *BufferPool
	enc  frameEncoding
}

func (c *FramedUDPConn) ReadMsgUDP(b, oob []byte) (n, oobn, flags int, addr *net.UDPAddr, err error) {
	n, oobn, flags, addr, err = c.UDPConn.ReadMsgUDP(b, oob)
	if err != nil {
		return 0, 0, 0, nil, err
	}
	n, err = c.enc.Decode(b[:n])
	return n, oobn, flags, addr, err
}

func (c *FramedUDPConn) WriteMsgUDP(b, oob []byte, addr *net.UDPAddr) (n, oobn int, err error) {
	t := c.pool.Get()
	defer c.pool.Put(t)

	n = c.enc.Encode(t, b)
	diff := n - len(b)
	n, oobn, err = c.UDPConn.WriteMsgUDP(t[:n], oob, addr)

	return max(n-diff, 0), oobn, err
}

func NewFramedBatchConn(conn BatchConn, pool *sync.Pool, opts FramedOpts) (c BatchConn, ok bool) {
	enc, ok := newFrameEncoding(opts)
	if !ok {
		return nil, false
	}

	return &FramedBatchConn{
		BatchConn: conn,
		pool:      WrapBufferPool(pool),
		enc:       enc,
	}, true
}

type FramedBatchConn struct {
	BatchConn
	pool *BufferPool
	enc  frameEncoding
}

func (c *FramedBatchConn) ReadBatch(ms []ipv4.Message, flags int) (n int, err error) {
	n, err = c.BatchConn.ReadBatch(ms, flags)
	if err != nil {
		return 0, err
	}

	for i := range ms[:n] {
		b := ms[i].Buffers[0][:ms[i].N]
		ms[i].N, err = c.enc.Decode(b)
		if err != nil {
			return 0, err
		}
	}

	return n, nil
}

func (c *FramedBatchConn) WriteBatch(ms []ipv4.Message, flags int) (n int, err error) {
	var inline [128][]byte
	pooled := inline[:0]
	if len(ms) > len(inline) {
		pooled = make([][]byte, 0, len(ms))
	}

	for i := range ms {
		t := c.pool.Get()
		pooled = append(pooled, t)

		n = c.enc.Encode(t, ms[i].Buffers[0])
		ms[i].Buffers[0] = t[:n]
	}

	// ms[i].N has incorrect N because the original data was modifier above
	// however, WG does not check this field, so this is fine
	n, err = c.BatchConn.WriteBatch(ms, flags)
	for _, buf := range pooled {
		c.pool.Put(buf)
	}
	return n, err
}
