package conceal

import "testing"

func BenchmarkUDPRawConn(b *testing.B) {
	benchmarkUseDeterministicRand(b)
	if !benchmarkFullMatrixEnabled() {
		readPayloads := [][]byte{
			benchmarkPayloads.initiation,
			benchmarkPayloads.transportSmall,
		}
		b.Run("Read/mixed", func(b *testing.B) {
			conn := newBenchmarkUDPConn(readPayloads)
			buf := make([]byte, benchmarkMaxPacketSize)
			benchmarkRunLoop(b, benchmarkAverageBytes(readPayloads...), nil, func() error {
				_, _, _, _, err := conn.ReadMsgUDP(buf, nil)
				return err
			})
		})

		writePayloads := [][]byte{
			benchmarkPayloads.initiation,
			benchmarkPayloads.transportSmall,
			benchmarkPayloads.transportMTU,
		}
		b.Run("Write/mixed", func(b *testing.B) {
			conn := newBenchmarkUDPConn(nil)
			next := 0
			benchmarkRunLoop(b, benchmarkAverageBytes(writePayloads...), nil, func() error {
				payload := writePayloads[next]
				next++
				if next == len(writePayloads) {
					next = 0
				}
				_, _, err := conn.WriteMsgUDP(payload, nil, benchmarkUDPAddr)
				return err
			})
		})
		return
	}

	readCases := []struct {
		name    string
		payload []byte
	}{
		{name: "Read/initiation", payload: benchmarkPayloads.initiation},
		{name: "Read/transport_small", payload: benchmarkPayloads.transportSmall},
	}
	for _, tc := range readCases {
		b.Run(tc.name, func(b *testing.B) {
			conn := newBenchmarkUDPConn([][]byte{tc.payload})
			buf := make([]byte, benchmarkMaxPacketSize)
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, _, _, _, err := conn.ReadMsgUDP(buf, nil)
				return err
			})
		})
	}

	writeCases := []struct {
		name    string
		payload []byte
	}{
		{name: "Write/initiation", payload: benchmarkPayloads.initiation},
		{name: "Write/transport_small", payload: benchmarkPayloads.transportSmall},
		{name: "Write/transport_mtu", payload: benchmarkPayloads.transportMTU},
	}
	for _, tc := range writeCases {
		b.Run(tc.name, func(b *testing.B) {
			conn := newBenchmarkUDPConn(nil)
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, _, err := conn.WriteMsgUDP(tc.payload, nil, benchmarkUDPAddr)
				return err
			})
		})
	}
}

func BenchmarkUDPMasquerade(b *testing.B) {
	benchmarkUseDeterministicRand(b)
	if !benchmarkFullMatrixEnabled() {
		readPayloads := [][]byte{
			benchmarkPayloads.initiation,
			benchmarkPayloads.transportSmall,
		}
		encoded := make([][]byte, len(readPayloads))
		for i, payload := range readPayloads {
			encoded[i] = benchmarkEncodeMasqueradeRecord(benchmarkMasqueradeRules, payload)
		}
		b.Run("Read/mixed", func(b *testing.B) {
			source := newBenchmarkUDPConn(encoded)
			pool := benchmarkNewBufferPool()
			conn, ok := NewMasqueradeUDPConn(source, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected udp masquerade benchmark conn")
			}
			buf := make([]byte, benchmarkMaxPacketSize)
			benchmarkRunLoop(b, benchmarkAverageBytes(readPayloads...), nil, func() error {
				_, _, _, _, err := conn.ReadMsgUDP(buf, nil)
				return err
			})
		})

		writePayloads := [][]byte{
			benchmarkPayloads.initiation,
			benchmarkPayloads.transportSmall,
			benchmarkPayloads.transportMTU,
		}
		b.Run("Write/mixed", func(b *testing.B) {
			sink := newBenchmarkUDPConn(nil)
			pool := benchmarkNewBufferPool()
			conn, ok := NewMasqueradeUDPConn(sink, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected udp masquerade benchmark conn")
			}
			next := 0
			benchmarkRunLoop(b, benchmarkAverageBytes(writePayloads...), nil, func() error {
				payload := writePayloads[next]
				next++
				if next == len(writePayloads) {
					next = 0
				}
				_, _, err := conn.WriteMsgUDP(payload, nil, benchmarkUDPAddr)
				return err
			})
		})
		return
	}

	readCases := []struct {
		name    string
		payload []byte
	}{
		{name: "Read/initiation", payload: benchmarkPayloads.initiation},
		{name: "Read/transport_small", payload: benchmarkPayloads.transportSmall},
	}
	for _, tc := range readCases {
		b.Run(tc.name, func(b *testing.B) {
			source := newBenchmarkUDPConn([][]byte{benchmarkEncodeMasqueradeRecord(benchmarkMasqueradeRules, tc.payload)})
			pool := benchmarkNewBufferPool()
			conn, ok := NewMasqueradeUDPConn(source, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected udp masquerade benchmark conn")
			}
			buf := make([]byte, benchmarkMaxPacketSize)
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, _, _, _, err := conn.ReadMsgUDP(buf, nil)
				return err
			})
		})
	}

	writeCases := []struct {
		name    string
		payload []byte
	}{
		{name: "Write/initiation", payload: benchmarkPayloads.initiation},
		{name: "Write/transport_small", payload: benchmarkPayloads.transportSmall},
		{name: "Write/transport_mtu", payload: benchmarkPayloads.transportMTU},
	}
	for _, tc := range writeCases {
		b.Run(tc.name, func(b *testing.B) {
			sink := newBenchmarkUDPConn(nil)
			pool := benchmarkNewBufferPool()
			conn, ok := NewMasqueradeUDPConn(sink, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected udp masquerade benchmark conn")
			}
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, _, err := conn.WriteMsgUDP(tc.payload, nil, benchmarkUDPAddr)
				return err
			})
		})
	}
}

func BenchmarkUDPFramed(b *testing.B) {
	benchmarkUseDeterministicRand(b)
	if !benchmarkFullMatrixEnabled() {
		compatOffPayloads := [][]byte{
			benchmarkPayloads.initiation,
			benchmarkPayloads.response,
			benchmarkPayloads.cookie,
			benchmarkPayloads.transportKeepalive,
			benchmarkPayloads.transportMTU,
		}
		compatOffEncoded := make([][]byte, len(compatOffPayloads))
		for i, payload := range compatOffPayloads {
			compatOffEncoded[i] = benchmarkEncodeFramedRecord(benchmarkFramedOpts, payload)
		}
		b.Run("Read/compat_off/mixed", func(b *testing.B) {
			source := newBenchmarkUDPConn(compatOffEncoded)
			pool := benchmarkNewBufferPool()
			conn, ok := NewFramedUDPConn(source, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected framed udp benchmark conn")
			}
			buf := make([]byte, benchmarkMaxPacketSize)
			benchmarkRunLoop(b, benchmarkAverageBytes(compatOffPayloads...), nil, func() error {
				_, _, _, _, err := conn.ReadMsgUDP(buf, nil)
				return err
			})
		})
		b.Run("Write/compat_off/mixed", func(b *testing.B) {
			sink := newBenchmarkUDPConn(nil)
			pool := benchmarkNewBufferPool()
			conn, ok := NewFramedUDPConn(sink, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected framed udp benchmark conn")
			}
			next := 0
			benchmarkRunLoop(b, benchmarkAverageBytes(compatOffPayloads...), nil, func() error {
				payload := compatOffPayloads[next]
				next++
				if next == len(compatOffPayloads) {
					next = 0
				}
				_, _, err := conn.WriteMsgUDP(payload, nil, benchmarkUDPAddr)
				return err
			})
		})

		compatOnPayloads := [][]byte{
			benchmarkPayloads.compatInitiation,
			benchmarkPayloads.compatTransportMTU,
		}
		compatOnEncoded := make([][]byte, len(compatOnPayloads))
		for i, payload := range compatOnPayloads {
			compatOnEncoded[i] = benchmarkEncodeFramedRecord(benchmarkFramedCompatOpts, payload)
		}
		b.Run("Read/compat_on/mixed", func(b *testing.B) {
			source := newBenchmarkUDPConn(compatOnEncoded)
			pool := benchmarkNewBufferPool()
			conn, ok := NewFramedUDPConn(source, pool, benchmarkFramedCompatOpts)
			if !ok {
				b.Fatal("expected compat framed udp benchmark conn")
			}
			buf := make([]byte, benchmarkMaxPacketSize)
			benchmarkRunLoop(b, benchmarkAverageBytes(compatOnPayloads...), nil, func() error {
				_, _, _, _, err := conn.ReadMsgUDP(buf, nil)
				return err
			})
		})
		b.Run("Write/compat_on/mixed", func(b *testing.B) {
			sink := newBenchmarkUDPConn(nil)
			pool := benchmarkNewBufferPool()
			conn, ok := NewFramedUDPConn(sink, pool, benchmarkFramedCompatOpts)
			if !ok {
				b.Fatal("expected compat framed udp benchmark conn")
			}
			next := 0
			benchmarkRunLoop(b, benchmarkAverageBytes(compatOnPayloads...), nil, func() error {
				payload := compatOnPayloads[next]
				next++
				if next == len(compatOnPayloads) {
					next = 0
				}
				_, _, err := conn.WriteMsgUDP(payload, nil, benchmarkUDPAddr)
				return err
			})
		})
		return
	}

	readCompatOff := []struct {
		name    string
		payload []byte
	}{
		{name: "Read/compat_off/initiation", payload: benchmarkPayloads.initiation},
		{name: "Read/compat_off/response", payload: benchmarkPayloads.response},
		{name: "Read/compat_off/cookie", payload: benchmarkPayloads.cookie},
		{name: "Read/compat_off/transport_keepalive", payload: benchmarkPayloads.transportKeepalive},
		{name: "Read/compat_off/transport_mtu", payload: benchmarkPayloads.transportMTU},
	}
	for _, tc := range readCompatOff {
		b.Run(tc.name, func(b *testing.B) {
			source := newBenchmarkUDPConn([][]byte{benchmarkEncodeFramedRecord(benchmarkFramedOpts, tc.payload)})
			pool := benchmarkNewBufferPool()
			conn, ok := NewFramedUDPConn(source, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected framed udp benchmark conn")
			}
			buf := make([]byte, benchmarkMaxPacketSize)
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, _, _, _, err := conn.ReadMsgUDP(buf, nil)
				return err
			})
		})
	}

	writeCompatOff := []struct {
		name    string
		payload []byte
	}{
		{name: "Write/compat_off/initiation", payload: benchmarkPayloads.initiation},
		{name: "Write/compat_off/response", payload: benchmarkPayloads.response},
		{name: "Write/compat_off/cookie", payload: benchmarkPayloads.cookie},
		{name: "Write/compat_off/transport_keepalive", payload: benchmarkPayloads.transportKeepalive},
		{name: "Write/compat_off/transport_mtu", payload: benchmarkPayloads.transportMTU},
	}
	for _, tc := range writeCompatOff {
		b.Run(tc.name, func(b *testing.B) {
			sink := newBenchmarkUDPConn(nil)
			pool := benchmarkNewBufferPool()
			conn, ok := NewFramedUDPConn(sink, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected framed udp benchmark conn")
			}
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, _, err := conn.WriteMsgUDP(tc.payload, nil, benchmarkUDPAddr)
				return err
			})
		})
	}

	readCompatOn := []struct {
		name    string
		payload []byte
	}{
		{name: "Read/compat_on/initiation", payload: benchmarkPayloads.compatInitiation},
		{name: "Read/compat_on/transport_mtu", payload: benchmarkPayloads.compatTransportMTU},
	}
	for _, tc := range readCompatOn {
		b.Run(tc.name, func(b *testing.B) {
			source := newBenchmarkUDPConn([][]byte{benchmarkEncodeFramedRecord(benchmarkFramedCompatOpts, tc.payload)})
			pool := benchmarkNewBufferPool()
			conn, ok := NewFramedUDPConn(source, pool, benchmarkFramedCompatOpts)
			if !ok {
				b.Fatal("expected compat framed udp benchmark conn")
			}
			buf := make([]byte, benchmarkMaxPacketSize)
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, _, _, _, err := conn.ReadMsgUDP(buf, nil)
				return err
			})
		})
	}

	writeCompatOn := []struct {
		name    string
		payload []byte
	}{
		{name: "Write/compat_on/initiation", payload: benchmarkPayloads.compatInitiation},
		{name: "Write/compat_on/transport_mtu", payload: benchmarkPayloads.compatTransportMTU},
	}
	for _, tc := range writeCompatOn {
		b.Run(tc.name, func(b *testing.B) {
			sink := newBenchmarkUDPConn(nil)
			pool := benchmarkNewBufferPool()
			conn, ok := NewFramedUDPConn(sink, pool, benchmarkFramedCompatOpts)
			if !ok {
				b.Fatal("expected compat framed udp benchmark conn")
			}
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, _, err := conn.WriteMsgUDP(tc.payload, nil, benchmarkUDPAddr)
				return err
			})
		})
	}
}

func BenchmarkUDPPrelude(b *testing.B) {
	benchmarkUseDeterministicRand(b)
	if !benchmarkFullMatrixEnabled() {
		b.Run("Write/initiation_mixed", func(b *testing.B) {
			fixtures := make([]*benchmarkUDPPreludeWriteFixture, benchmarkFixtureRingSize)
			for i := range fixtures {
				opts := benchmarkPreludeOneRule
				if i%2 == 1 {
					opts = benchmarkPreludeRulesPlusJunk
				}
				fixtures[i] = newBenchmarkUDPPreludeWriteFixture(opts, benchmarkPayloads.initiation)
			}
			benchmarkRunLoopWithFixtureRing(b, len(benchmarkPayloads.initiation), fixtures, nil, func(f *benchmarkUDPPreludeWriteFixture) error {
				return f.Write()
			})
		})

		b.Run("Write/passthrough_mixed", func(b *testing.B) {
			raw := newBenchmarkUDPConn(nil)
			pool := benchmarkNewBufferPool()
			conn, ok := NewPreludeUDPConn(raw, raw, pool, nil, benchmarkPreludeRulesPlusJunk)
			if !ok {
				b.Fatal("expected prelude udp benchmark conn")
			}
			payloads := [][]byte{
				benchmarkPayloads.transportSmall,
				benchmarkPayloads.transportMTU,
			}
			next := 0
			benchmarkRunLoop(b, benchmarkAverageBytes(payloads...), nil, func() error {
				payload := payloads[next]
				next++
				if next == len(payloads) {
					next = 0
				}
				_, _, err := conn.WriteMsgUDP(payload, nil, benchmarkUDPAddr)
				return err
			})
		})
		return
	}

	cases := []struct {
		name    string
		payload []byte
		opts    PreludeOpts
	}{
		{name: "Write/initiation/rules_only", payload: benchmarkPayloads.initiation, opts: benchmarkPreludeOneRule},
		{name: "Write/initiation/rules_plus_junk", payload: benchmarkPayloads.initiation, opts: benchmarkPreludeRulesPlusJunk},
		{name: "Write/transport_small/passthrough", payload: benchmarkPayloads.transportSmall, opts: benchmarkPreludeRulesPlusJunk},
		{name: "Write/transport_mtu/passthrough", payload: benchmarkPayloads.transportMTU, opts: benchmarkPreludeRulesPlusJunk},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			raw := newBenchmarkUDPConn(nil)
			pool := benchmarkNewBufferPool()
			conn, ok := NewPreludeUDPConn(raw, raw, pool, nil, tc.opts)
			if !ok {
				b.Fatal("expected prelude udp benchmark conn")
			}
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, _, err := conn.WriteMsgUDP(tc.payload, nil, benchmarkUDPAddr)
				return err
			})
		})
	}
}

func BenchmarkUDPPipeline(b *testing.B) {
	benchmarkUseDeterministicRand(b)
	if !benchmarkFullMatrixEnabled() {
		readPayloads := [][]byte{
			benchmarkPayloads.initiation,
			benchmarkPayloads.transportSmall,
		}
		encoded := make([][]byte, len(readPayloads))
		for i, payload := range readPayloads {
			encoded[i] = benchmarkEncodeMasqueradeRecord(
				benchmarkMasqueradeRules,
				benchmarkEncodeFramedRecord(benchmarkFramedOpts, payload),
			)
		}
		b.Run("Read/masquerade_framed/mixed", func(b *testing.B) {
			raw := newBenchmarkUDPConn(encoded)
			pool := benchmarkNewBufferPool()
			masquerade, ok := NewMasqueradeUDPConn(raw, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected masquerade udp benchmark conn")
			}
			framed, ok := NewFramedUDPConn(masquerade, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected framed udp benchmark conn")
			}
			conn, ok := NewPreludeUDPConn(framed, raw, pool, nil, benchmarkPreludeRulesPlusJunk)
			if !ok {
				b.Fatal("expected prelude udp benchmark conn")
			}
			buf := make([]byte, benchmarkMaxPacketSize)
			benchmarkRunLoop(b, benchmarkAverageBytes(readPayloads...), nil, func() error {
				_, _, _, _, err := conn.ReadMsgUDP(buf, nil)
				return err
			})
		})

		writePayloads := [][]byte{
			benchmarkPayloads.initiation,
			benchmarkPayloads.transportSmall,
			benchmarkPayloads.transportMTU,
		}
		b.Run("Write/full_pipeline/mixed", func(b *testing.B) {
			raw := newBenchmarkUDPConn(nil)
			pool := benchmarkNewBufferPool()
			masquerade, ok := NewMasqueradeUDPConn(raw, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected masquerade udp benchmark conn")
			}
			framed, ok := NewFramedUDPConn(masquerade, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected framed udp benchmark conn")
			}
			conn, ok := NewPreludeUDPConn(framed, raw, pool, nil, benchmarkPreludeRulesPlusJunk)
			if !ok {
				b.Fatal("expected prelude udp benchmark conn")
			}
			next := 0
			benchmarkRunLoop(b, benchmarkAverageBytes(writePayloads...), nil, func() error {
				payload := writePayloads[next]
				next++
				if next == len(writePayloads) {
					next = 0
				}
				_, _, err := conn.WriteMsgUDP(payload, nil, benchmarkUDPAddr)
				return err
			})
		})
		return
	}

	readCases := []struct {
		name    string
		payload []byte
	}{
		{name: "Read/masquerade_framed/initiation", payload: benchmarkPayloads.initiation},
		{name: "Read/masquerade_framed/transport_small", payload: benchmarkPayloads.transportSmall},
	}
	for _, tc := range readCases {
		b.Run(tc.name, func(b *testing.B) {
			raw := newBenchmarkUDPConn([][]byte{
				benchmarkEncodeMasqueradeRecord(
					benchmarkMasqueradeRules,
					benchmarkEncodeFramedRecord(benchmarkFramedOpts, tc.payload),
				),
			})
			pool := benchmarkNewBufferPool()
			masquerade, ok := NewMasqueradeUDPConn(raw, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected masquerade udp benchmark conn")
			}
			framed, ok := NewFramedUDPConn(masquerade, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected framed udp benchmark conn")
			}
			conn, ok := NewPreludeUDPConn(framed, raw, pool, nil, benchmarkPreludeRulesPlusJunk)
			if !ok {
				b.Fatal("expected prelude udp benchmark conn")
			}
			buf := make([]byte, benchmarkMaxPacketSize)
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, _, _, _, err := conn.ReadMsgUDP(buf, nil)
				return err
			})
		})
	}

	writeCases := []struct {
		name    string
		payload []byte
	}{
		{name: "Write/full_pipeline/initiation", payload: benchmarkPayloads.initiation},
		{name: "Write/full_pipeline/transport_small", payload: benchmarkPayloads.transportSmall},
		{name: "Write/full_pipeline/transport_mtu", payload: benchmarkPayloads.transportMTU},
	}
	for _, tc := range writeCases {
		b.Run(tc.name, func(b *testing.B) {
			raw := newBenchmarkUDPConn(nil)
			pool := benchmarkNewBufferPool()
			masquerade, ok := NewMasqueradeUDPConn(raw, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected masquerade udp benchmark conn")
			}
			framed, ok := NewFramedUDPConn(masquerade, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected framed udp benchmark conn")
			}
			conn, ok := NewPreludeUDPConn(framed, raw, pool, nil, benchmarkPreludeRulesPlusJunk)
			if !ok {
				b.Fatal("expected prelude udp benchmark conn")
			}
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, _, err := conn.WriteMsgUDP(tc.payload, nil, benchmarkUDPAddr)
				return err
			})
		})
	}
}

func BenchmarkUDPBatchRaw(b *testing.B) {
	benchmarkUseDeterministicRand(b)
	if !benchmarkFullMatrixEnabled() {
		for _, batchSize := range []int{8, 64} {
			name := benchmarkBatchName(batchSize)
			payloads := benchmarkRepeatPayload(benchmarkPayloads.transportSmall, batchSize)

			b.Run("Read/"+name+"/transport_small", func(b *testing.B) {
				conn := newBenchmarkBatchConn([][]benchmarkBatchPacket{benchmarkBatchPackets(payloads)})
				msgs := benchmarkNewBatchReadMessages(batchSize, 0)
				benchmarkRunLoop(b, benchmarkTotalBytes(payloads), nil, func() error {
					_, err := conn.ReadBatch(msgs, 0)
					return err
				})
			})

			b.Run("Write/"+name+"/transport_small", func(b *testing.B) {
				conn := newBenchmarkBatchConn(nil)
				fixture := newBenchmarkBatchWriteFixture(payloads, 0)
				benchmarkRunLoop(b, benchmarkTotalBytes(payloads), nil, func() error {
					_, err := conn.WriteBatch(fixture.msgs, 0)
					return err
				})
			})
		}
		return
	}

	for _, batchSize := range []int{1, 8, 64} {
		name := benchmarkBatchName(batchSize)
		payloads := benchmarkRepeatPayload(benchmarkPayloads.transportSmall, batchSize)

		b.Run("Read/"+name+"/transport_small", func(b *testing.B) {
			conn := newBenchmarkBatchConn([][]benchmarkBatchPacket{benchmarkBatchPackets(payloads)})
			msgs := benchmarkNewBatchReadMessages(batchSize, 0)
			benchmarkRunLoop(b, benchmarkTotalBytes(payloads), nil, func() error {
				_, err := conn.ReadBatch(msgs, 0)
				return err
			})
		})

		b.Run("Write/"+name+"/transport_small", func(b *testing.B) {
			conn := newBenchmarkBatchConn(nil)
			fixture := newBenchmarkBatchWriteFixture(payloads, 0)
			benchmarkRunLoop(b, benchmarkTotalBytes(payloads), nil, func() error {
				_, err := conn.WriteBatch(fixture.msgs, 0)
				return err
			})
		})
	}
}

func BenchmarkUDPBatchMasquerade(b *testing.B) {
	benchmarkUseDeterministicRand(b)
	if !benchmarkFullMatrixEnabled() {
		for _, batchSize := range []int{8, 64} {
			name := benchmarkBatchName(batchSize)
			payloads := benchmarkRepeatPayload(benchmarkPayloads.transportSmall, batchSize)
			encoded := make([][]byte, len(payloads))
			for i, payload := range payloads {
				encoded[i] = benchmarkEncodeMasqueradeRecord(benchmarkMasqueradeRules, payload)
			}

			b.Run("Read/"+name+"/transport_small", func(b *testing.B) {
				raw := newBenchmarkBatchConn([][]benchmarkBatchPacket{benchmarkBatchPackets(encoded)})
				pool := benchmarkNewBufferPool()
				conn, ok := NewMasqueradeBatchConn(raw, pool, MasqueradeOpts{
					RulesIn:  benchmarkMasqueradeRules,
					RulesOut: benchmarkMasqueradeRules,
				})
				if !ok {
					b.Fatal("expected batch masquerade benchmark conn")
				}
				msgs := benchmarkNewBatchReadMessages(batchSize, 0)
				benchmarkRunLoop(b, benchmarkTotalBytes(payloads), nil, func() error {
					_, err := conn.ReadBatch(msgs, 0)
					return err
				})
			})
		}

		b.Run("Write/batch8/transport_small", func(b *testing.B) {
			payloads := benchmarkRepeatPayload(benchmarkPayloads.transportSmall, 8)
			raw := newBenchmarkBatchConn(nil)
			pool := benchmarkNewBufferPool()
			conn, ok := NewMasqueradeBatchConn(raw, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected batch masquerade benchmark conn")
			}
			fixtures := benchmarkNewBatchWriteFixtureRing([][][]byte{payloads}, 0)
			benchmarkRunLoopWithFixtureRing(b, benchmarkTotalBytes(payloads), fixtures, func(f *benchmarkBatchWriteFixture) {
				f.Reset()
			}, func(f *benchmarkBatchWriteFixture) error {
				_, err := conn.WriteBatch(f.msgs, 0)
				return err
			})
		})

		b.Run("Write/batch64/mixed_transport", func(b *testing.B) {
			small := benchmarkRepeatPayload(benchmarkPayloads.transportSmall, 64)
			mtu := benchmarkRepeatPayload(benchmarkPayloads.transportMTU, 64)
			raw := newBenchmarkBatchConn(nil)
			pool := benchmarkNewBufferPool()
			conn, ok := NewMasqueradeBatchConn(raw, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected batch masquerade benchmark conn")
			}
			fixtures := benchmarkNewBatchWriteFixtureRing([][][]byte{small, mtu}, 0)
			benchmarkRunLoopWithFixtureRing(
				b,
				benchmarkAverageInts(benchmarkTotalBytes(small), benchmarkTotalBytes(mtu)),
				fixtures,
				func(f *benchmarkBatchWriteFixture) { f.Reset() },
				func(f *benchmarkBatchWriteFixture) error {
					_, err := conn.WriteBatch(f.msgs, 0)
					return err
				},
			)
		})
		return
	}

	for _, batchSize := range []int{1, 8, 64} {
		name := benchmarkBatchName(batchSize)
		payloads := benchmarkRepeatPayload(benchmarkPayloads.transportSmall, batchSize)
		encoded := make([][]byte, len(payloads))
		for i, payload := range payloads {
			encoded[i] = benchmarkEncodeMasqueradeRecord(benchmarkMasqueradeRules, payload)
		}

		b.Run("Read/"+name+"/transport_small", func(b *testing.B) {
			raw := newBenchmarkBatchConn([][]benchmarkBatchPacket{benchmarkBatchPackets(encoded)})
			pool := benchmarkNewBufferPool()
			conn, ok := NewMasqueradeBatchConn(raw, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected batch masquerade benchmark conn")
			}
			msgs := benchmarkNewBatchReadMessages(batchSize, 0)
			benchmarkRunLoop(b, benchmarkTotalBytes(payloads), nil, func() error {
				_, err := conn.ReadBatch(msgs, 0)
				return err
			})
		})

		b.Run("Write/"+name+"/transport_small", func(b *testing.B) {
			raw := newBenchmarkBatchConn(nil)
			pool := benchmarkNewBufferPool()
			conn, ok := NewMasqueradeBatchConn(raw, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected batch masquerade benchmark conn")
			}
			fixture := newBenchmarkBatchWriteFixture(payloads, 0)
			benchmarkRunLoop(b, benchmarkTotalBytes(payloads), func() {
				fixture.Reset()
			}, func() error {
				_, err := conn.WriteBatch(fixture.msgs, 0)
				return err
			})
		})
	}

	b.Run("Write/batch64/transport_mtu", func(b *testing.B) {
		payloads := benchmarkRepeatPayload(benchmarkPayloads.transportMTU, 64)
		raw := newBenchmarkBatchConn(nil)
		pool := benchmarkNewBufferPool()
		conn, ok := NewMasqueradeBatchConn(raw, pool, MasqueradeOpts{
			RulesIn:  benchmarkMasqueradeRules,
			RulesOut: benchmarkMasqueradeRules,
		})
		if !ok {
			b.Fatal("expected batch masquerade benchmark conn")
		}
		fixture := newBenchmarkBatchWriteFixture(payloads, 0)
		benchmarkRunLoop(b, benchmarkTotalBytes(payloads), func() {
			fixture.Reset()
		}, func() error {
			_, err := conn.WriteBatch(fixture.msgs, 0)
			return err
		})
	})
}

func BenchmarkUDPBatchFramed(b *testing.B) {
	benchmarkUseDeterministicRand(b)
	if !benchmarkFullMatrixEnabled() {
		for _, batchSize := range []int{8, 64} {
			name := benchmarkBatchName(batchSize)
			payloads := benchmarkRepeatPayload(benchmarkPayloads.transportSmall, batchSize)
			encoded := make([][]byte, len(payloads))
			for i, payload := range payloads {
				encoded[i] = benchmarkEncodeFramedRecord(benchmarkFramedOpts, payload)
			}

			b.Run("Read/"+name+"/compat_off/transport_small", func(b *testing.B) {
				raw := newBenchmarkBatchConn([][]benchmarkBatchPacket{benchmarkBatchPackets(encoded)})
				pool := benchmarkNewBufferPool()
				conn, ok := NewFramedBatchConn(raw, pool, benchmarkFramedOpts)
				if !ok {
					b.Fatal("expected batch framed benchmark conn")
				}
				msgs := benchmarkNewBatchReadMessages(batchSize, 0)
				benchmarkRunLoop(b, benchmarkTotalBytes(payloads), nil, func() error {
					_, err := conn.ReadBatch(msgs, 0)
					return err
				})
			})

			b.Run("Write/"+name+"/compat_off/transport_small", func(b *testing.B) {
				raw := newBenchmarkBatchConn(nil)
				pool := benchmarkNewBufferPool()
				conn, ok := NewFramedBatchConn(raw, pool, benchmarkFramedOpts)
				if !ok {
					b.Fatal("expected batch framed benchmark conn")
				}
				fixtures := benchmarkNewBatchWriteFixtureRing([][][]byte{payloads}, 0)
				benchmarkRunLoopWithFixtureRing(b, benchmarkTotalBytes(payloads), fixtures, func(f *benchmarkBatchWriteFixture) {
					f.Reset()
				}, func(f *benchmarkBatchWriteFixture) error {
					_, err := conn.WriteBatch(f.msgs, 0)
					return err
				})
			})
		}

		b.Run("Read/batch8/compat_on/initiation", func(b *testing.B) {
			payloads := benchmarkRepeatPayload(benchmarkPayloads.compatInitiation, 8)
			encoded := make([][]byte, len(payloads))
			for i, payload := range payloads {
				encoded[i] = benchmarkEncodeFramedRecord(benchmarkFramedCompatOpts, payload)
			}
			raw := newBenchmarkBatchConn([][]benchmarkBatchPacket{benchmarkBatchPackets(encoded)})
			pool := benchmarkNewBufferPool()
			conn, ok := NewFramedBatchConn(raw, pool, benchmarkFramedCompatOpts)
			if !ok {
				b.Fatal("expected compat batch framed benchmark conn")
			}
			msgs := benchmarkNewBatchReadMessages(8, 0)
			benchmarkRunLoop(b, benchmarkTotalBytes(payloads), nil, func() error {
				_, err := conn.ReadBatch(msgs, 0)
				return err
			})
		})

		b.Run("Write/batch8/compat_on/initiation", func(b *testing.B) {
			payloads := benchmarkRepeatPayload(benchmarkPayloads.compatInitiation, 8)
			raw := newBenchmarkBatchConn(nil)
			pool := benchmarkNewBufferPool()
			conn, ok := NewFramedBatchConn(raw, pool, benchmarkFramedCompatOpts)
			if !ok {
				b.Fatal("expected compat batch framed benchmark conn")
			}
			fixtures := benchmarkNewBatchWriteFixtureRing([][][]byte{payloads}, 0)
			benchmarkRunLoopWithFixtureRing(b, benchmarkTotalBytes(payloads), fixtures, func(f *benchmarkBatchWriteFixture) {
				f.Reset()
			}, func(f *benchmarkBatchWriteFixture) error {
				_, err := conn.WriteBatch(f.msgs, 0)
				return err
			})
		})
		return
	}

	for _, batchSize := range []int{1, 8, 64} {
		name := benchmarkBatchName(batchSize)
		payloads := benchmarkRepeatPayload(benchmarkPayloads.transportSmall, batchSize)
		encoded := make([][]byte, len(payloads))
		for i, payload := range payloads {
			encoded[i] = benchmarkEncodeFramedRecord(benchmarkFramedOpts, payload)
		}

		b.Run("Read/"+name+"/compat_off/transport_small", func(b *testing.B) {
			raw := newBenchmarkBatchConn([][]benchmarkBatchPacket{benchmarkBatchPackets(encoded)})
			pool := benchmarkNewBufferPool()
			conn, ok := NewFramedBatchConn(raw, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected batch framed benchmark conn")
			}
			msgs := benchmarkNewBatchReadMessages(batchSize, 0)
			benchmarkRunLoop(b, benchmarkTotalBytes(payloads), nil, func() error {
				_, err := conn.ReadBatch(msgs, 0)
				return err
			})
		})

		b.Run("Write/"+name+"/compat_off/transport_small", func(b *testing.B) {
			raw := newBenchmarkBatchConn(nil)
			pool := benchmarkNewBufferPool()
			conn, ok := NewFramedBatchConn(raw, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected batch framed benchmark conn")
			}
			fixture := newBenchmarkBatchWriteFixture(payloads, 0)
			benchmarkRunLoop(b, benchmarkTotalBytes(payloads), func() {
				fixture.Reset()
			}, func() error {
				_, err := conn.WriteBatch(fixture.msgs, 0)
				return err
			})
		})
	}

	b.Run("Read/batch8/compat_on/initiation", func(b *testing.B) {
		payloads := benchmarkRepeatPayload(benchmarkPayloads.compatInitiation, 8)
		encoded := make([][]byte, len(payloads))
		for i, payload := range payloads {
			encoded[i] = benchmarkEncodeFramedRecord(benchmarkFramedCompatOpts, payload)
		}
		raw := newBenchmarkBatchConn([][]benchmarkBatchPacket{benchmarkBatchPackets(encoded)})
		pool := benchmarkNewBufferPool()
		conn, ok := NewFramedBatchConn(raw, pool, benchmarkFramedCompatOpts)
		if !ok {
			b.Fatal("expected compat batch framed benchmark conn")
		}
		msgs := benchmarkNewBatchReadMessages(8, 0)
		benchmarkRunLoop(b, benchmarkTotalBytes(payloads), nil, func() error {
			_, err := conn.ReadBatch(msgs, 0)
			return err
		})
	})

	b.Run("Write/batch8/compat_on/initiation", func(b *testing.B) {
		payloads := benchmarkRepeatPayload(benchmarkPayloads.compatInitiation, 8)
		raw := newBenchmarkBatchConn(nil)
		pool := benchmarkNewBufferPool()
		conn, ok := NewFramedBatchConn(raw, pool, benchmarkFramedCompatOpts)
		if !ok {
			b.Fatal("expected compat batch framed benchmark conn")
		}
		fixture := newBenchmarkBatchWriteFixture(payloads, 0)
		benchmarkRunLoop(b, benchmarkTotalBytes(payloads), func() {
			fixture.Reset()
		}, func() error {
			_, err := conn.WriteBatch(fixture.msgs, 0)
			return err
		})
	})
}

func BenchmarkUDPBatchPrelude(b *testing.B) {
	benchmarkUseDeterministicRand(b)
	if !benchmarkFullMatrixEnabled() {
		b.Run("Write/batch8/initiation_present", func(b *testing.B) {
			payloads := benchmarkInitiationBatch(8)
			raw := newBenchmarkBatchConn(nil)
			pool := benchmarkNewBufferPool()
			msgsPool := benchmarkNewMsgsPool()
			conn, ok := NewPreludeBatchConn(raw, raw, pool, msgsPool, nil, benchmarkPreludeRulesPlusJunk)
			if !ok {
				b.Fatal("expected batch prelude benchmark conn")
			}
			fixtures := benchmarkNewBatchWriteFixtureRing([][][]byte{payloads}, 0)
			benchmarkRunLoopWithFixtureRing(b, benchmarkTotalBytes(payloads), fixtures, func(f *benchmarkBatchWriteFixture) {
				f.Reset()
			}, func(f *benchmarkBatchWriteFixture) error {
				_, err := conn.WriteBatch(f.msgs, 0)
				return err
			})
		})

		b.Run("Write/batch64/mixed_initiation_presence", func(b *testing.B) {
			withInitiation := benchmarkInitiationBatch(64)
			noInitiation := benchmarkRepeatPayload(benchmarkPayloads.transportSmall, 64)
			raw := newBenchmarkBatchConn(nil)
			pool := benchmarkNewBufferPool()
			msgsPool := benchmarkNewMsgsPool()
			conn, ok := NewPreludeBatchConn(raw, raw, pool, msgsPool, nil, benchmarkPreludeRulesPlusJunk)
			if !ok {
				b.Fatal("expected batch prelude benchmark conn")
			}
			fixtures := benchmarkNewBatchWriteFixtureRing([][][]byte{withInitiation, noInitiation}, 0)
			benchmarkRunLoopWithFixtureRing(
				b,
				benchmarkAverageInts(benchmarkTotalBytes(withInitiation), benchmarkTotalBytes(noInitiation)),
				fixtures,
				func(f *benchmarkBatchWriteFixture) { f.Reset() },
				func(f *benchmarkBatchWriteFixture) error {
					_, err := conn.WriteBatch(f.msgs, 0)
					return err
				},
			)
		})
		return
	}

	for _, batchSize := range []int{1, 8, 64} {
		name := benchmarkBatchName(batchSize)
		payloads := benchmarkInitiationBatch(batchSize)
		raw := newBenchmarkBatchConn(nil)
		pool := benchmarkNewBufferPool()
		msgsPool := benchmarkNewMsgsPool()
		conn, ok := NewPreludeBatchConn(raw, raw, pool, msgsPool, nil, benchmarkPreludeRulesPlusJunk)
		if !ok {
			b.Fatal("expected batch prelude benchmark conn")
		}
		fixture := newBenchmarkBatchWriteFixture(payloads, 0)
		b.Run("Write/"+name+"/initiation_present", func(b *testing.B) {
			benchmarkRunLoop(b, benchmarkTotalBytes(payloads), nil, func() error {
				_, err := conn.WriteBatch(fixture.msgs, 0)
				return err
			})
		})
	}

	for _, batchSize := range []int{8, 64} {
		name := benchmarkBatchName(batchSize)
		payloads := benchmarkRepeatPayload(benchmarkPayloads.transportSmall, batchSize)
		raw := newBenchmarkBatchConn(nil)
		pool := benchmarkNewBufferPool()
		msgsPool := benchmarkNewMsgsPool()
		conn, ok := NewPreludeBatchConn(raw, raw, pool, msgsPool, nil, benchmarkPreludeRulesPlusJunk)
		if !ok {
			b.Fatal("expected batch prelude benchmark conn")
		}
		fixture := newBenchmarkBatchWriteFixture(payloads, 0)
		b.Run("Write/"+name+"/no_initiation", func(b *testing.B) {
			benchmarkRunLoop(b, benchmarkTotalBytes(payloads), nil, func() error {
				_, err := conn.WriteBatch(fixture.msgs, 0)
				return err
			})
		})
	}
}

func BenchmarkUDPBatchPipeline(b *testing.B) {
	benchmarkUseDeterministicRand(b)
	if !benchmarkFullMatrixEnabled() {
		for _, batchSize := range []int{8, 64} {
			name := benchmarkBatchName(batchSize)
			payloads := benchmarkRepeatPayload(benchmarkPayloads.transportSmall, batchSize)
			encoded := make([][]byte, len(payloads))
			for i, payload := range payloads {
				encoded[i] = benchmarkEncodeMasqueradeRecord(
					benchmarkMasqueradeRules,
					benchmarkEncodeFramedRecord(benchmarkFramedOpts, payload),
				)
			}
			b.Run("Read/"+name+"/masquerade_framed/transport_small", func(b *testing.B) {
				raw := newBenchmarkBatchConn([][]benchmarkBatchPacket{benchmarkBatchPackets(encoded)})
				pool := benchmarkNewBufferPool()
				msgsPool := benchmarkNewMsgsPool()
				masquerade, ok := NewMasqueradeBatchConn(raw, pool, MasqueradeOpts{
					RulesIn:  benchmarkMasqueradeRules,
					RulesOut: benchmarkMasqueradeRules,
				})
				if !ok {
					b.Fatal("expected batch masquerade benchmark conn")
				}
				framed, ok := NewFramedBatchConn(masquerade, pool, benchmarkFramedOpts)
				if !ok {
					b.Fatal("expected batch framed benchmark conn")
				}
				conn, ok := NewPreludeBatchConn(framed, raw, pool, msgsPool, nil, benchmarkPreludeRulesPlusJunk)
				if !ok {
					b.Fatal("expected batch pipeline benchmark conn")
				}
				msgs := benchmarkNewBatchReadMessages(batchSize, 0)
				benchmarkRunLoop(b, benchmarkTotalBytes(payloads), nil, func() error {
					_, err := conn.ReadBatch(msgs, 0)
					return err
				})
			})
		}

		b.Run("Write/batch8/full_pipeline/initiation_present", func(b *testing.B) {
			payloads := benchmarkInitiationBatch(8)
			raw := newBenchmarkBatchConn(nil)
			pool := benchmarkNewBufferPool()
			msgsPool := benchmarkNewMsgsPool()
			masquerade, ok := NewMasqueradeBatchConn(raw, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected batch masquerade benchmark conn")
			}
			framed, ok := NewFramedBatchConn(masquerade, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected batch framed benchmark conn")
			}
			conn, ok := NewPreludeBatchConn(framed, raw, pool, msgsPool, nil, benchmarkPreludeRulesPlusJunk)
			if !ok {
				b.Fatal("expected batch pipeline benchmark conn")
			}
			fixtures := benchmarkNewBatchWriteFixtureRing([][][]byte{payloads}, 0)
			benchmarkRunLoopWithFixtureRing(b, benchmarkTotalBytes(payloads), fixtures, func(f *benchmarkBatchWriteFixture) {
				f.Reset()
			}, func(f *benchmarkBatchWriteFixture) error {
				_, err := conn.WriteBatch(f.msgs, 0)
				return err
			})
		})

		b.Run("Write/batch64/full_pipeline/mixed_initiation_presence", func(b *testing.B) {
			withInitiation := benchmarkInitiationBatch(64)
			noInitiation := benchmarkRepeatPayload(benchmarkPayloads.transportSmall, 64)
			raw := newBenchmarkBatchConn(nil)
			pool := benchmarkNewBufferPool()
			msgsPool := benchmarkNewMsgsPool()
			masquerade, ok := NewMasqueradeBatchConn(raw, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected batch masquerade benchmark conn")
			}
			framed, ok := NewFramedBatchConn(masquerade, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected batch framed benchmark conn")
			}
			conn, ok := NewPreludeBatchConn(framed, raw, pool, msgsPool, nil, benchmarkPreludeRulesPlusJunk)
			if !ok {
				b.Fatal("expected batch pipeline benchmark conn")
			}
			fixtures := benchmarkNewBatchWriteFixtureRing([][][]byte{withInitiation, noInitiation}, 0)
			benchmarkRunLoopWithFixtureRing(
				b,
				benchmarkAverageInts(benchmarkTotalBytes(withInitiation), benchmarkTotalBytes(noInitiation)),
				fixtures,
				func(f *benchmarkBatchWriteFixture) { f.Reset() },
				func(f *benchmarkBatchWriteFixture) error {
					_, err := conn.WriteBatch(f.msgs, 0)
					return err
				},
			)
		})
		return
	}

	for _, batchSize := range []int{8, 64} {
		name := benchmarkBatchName(batchSize)
		payloads := benchmarkRepeatPayload(benchmarkPayloads.transportSmall, batchSize)
		encoded := make([][]byte, len(payloads))
		for i, payload := range payloads {
			encoded[i] = benchmarkEncodeMasqueradeRecord(
				benchmarkMasqueradeRules,
				benchmarkEncodeFramedRecord(benchmarkFramedOpts, payload),
			)
		}
		b.Run("Read/"+name+"/masquerade_framed/transport_small", func(b *testing.B) {
			raw := newBenchmarkBatchConn([][]benchmarkBatchPacket{benchmarkBatchPackets(encoded)})
			pool := benchmarkNewBufferPool()
			msgsPool := benchmarkNewMsgsPool()
			masquerade, ok := NewMasqueradeBatchConn(raw, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected batch masquerade benchmark conn")
			}
			framed, ok := NewFramedBatchConn(masquerade, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected batch framed benchmark conn")
			}
			conn, ok := NewPreludeBatchConn(framed, raw, pool, msgsPool, nil, benchmarkPreludeRulesPlusJunk)
			if !ok {
				b.Fatal("expected batch pipeline benchmark conn")
			}
			msgs := benchmarkNewBatchReadMessages(batchSize, 0)
			benchmarkRunLoop(b, benchmarkTotalBytes(payloads), nil, func() error {
				_, err := conn.ReadBatch(msgs, 0)
				return err
			})
		})
	}

	writeCases := []struct {
		name     string
		payloads [][]byte
	}{
		{name: "Write/batch8/full_pipeline/initiation_present", payloads: benchmarkInitiationBatch(8)},
		{name: "Write/batch64/full_pipeline/initiation_present", payloads: benchmarkInitiationBatch(64)},
		{name: "Write/batch64/full_pipeline/no_initiation", payloads: benchmarkRepeatPayload(benchmarkPayloads.transportSmall, 64)},
	}
	for _, tc := range writeCases {
		b.Run(tc.name, func(b *testing.B) {
			raw := newBenchmarkBatchConn(nil)
			pool := benchmarkNewBufferPool()
			msgsPool := benchmarkNewMsgsPool()
			masquerade, ok := NewMasqueradeBatchConn(raw, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected batch masquerade benchmark conn")
			}
			framed, ok := NewFramedBatchConn(masquerade, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected batch framed benchmark conn")
			}
			conn, ok := NewPreludeBatchConn(framed, raw, pool, msgsPool, nil, benchmarkPreludeRulesPlusJunk)
			if !ok {
				b.Fatal("expected batch pipeline benchmark conn")
			}
			fixture := newBenchmarkBatchWriteFixture(tc.payloads, 0)
			benchmarkRunLoop(b, benchmarkTotalBytes(tc.payloads), func() {
				fixture.Reset()
			}, func() error {
				_, err := conn.WriteBatch(fixture.msgs, 0)
				return err
			})
		})
	}
}

func benchmarkBatchName(batchSize int) string {
	switch batchSize {
	case 1:
		return "batch1"
	case 8:
		return "batch8"
	case 64:
		return "batch64"
	default:
		panic("unsupported benchmark batch size")
	}
}

type benchmarkUDPPreludeWriteFixture struct {
	conn    *PreludeUDPConn
	payload []byte
}

func newBenchmarkUDPPreludeWriteFixture(opts PreludeOpts, payload []byte) *benchmarkUDPPreludeWriteFixture {
	raw := newBenchmarkUDPConn(nil)
	pool := benchmarkNewBufferPool()
	conn, ok := NewPreludeUDPConn(raw, raw, pool, nil, opts)
	if !ok {
		panic("expected prelude udp benchmark conn")
	}
	return &benchmarkUDPPreludeWriteFixture{
		conn:    conn,
		payload: payload,
	}
}

func (f *benchmarkUDPPreludeWriteFixture) Write() error {
	_, _, err := f.conn.WriteMsgUDP(f.payload, nil, benchmarkUDPAddr)
	return err
}

func benchmarkNewBatchWriteFixtureRing(payloadSets [][][]byte, oobCap int) []*benchmarkBatchWriteFixture {
	if len(payloadSets) == 0 {
		panic("benchmark batch payload sets must not be empty")
	}
	fixtures := make([]*benchmarkBatchWriteFixture, benchmarkFixtureRingSize)
	for i := range fixtures {
		fixtures[i] = newBenchmarkBatchWriteFixture(payloadSets[i%len(payloadSets)], oobCap)
	}
	return fixtures
}
