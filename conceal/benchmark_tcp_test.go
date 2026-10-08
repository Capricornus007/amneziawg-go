package conceal

import "testing"

func BenchmarkTCPRawConn(b *testing.B) {
	benchmarkUseDeterministicRand(b)
	if benchmarkFullMatrixEnabled() {
		benchmarkTCPRawConnFull(b)
		return
	}

	readPayloads := [][]byte{
		benchmarkPayloads.initiation,
		benchmarkPayloads.transportSmall,
	}
	b.Run("Read/mixed", func(b *testing.B) {
		conn := newBenchmarkStreamChunksConn(readPayloads)
		buf := make([]byte, benchmarkMaxPacketSize)
		benchmarkRunLoop(b, benchmarkAverageBytes(readPayloads...), nil, func() error {
			_, err := conn.Read(buf)
			return err
		})
	})

	writePayloads := [][]byte{
		benchmarkPayloads.initiation,
		benchmarkPayloads.transportSmall,
		benchmarkPayloads.transportMTU,
	}
	b.Run("Write/mixed", func(b *testing.B) {
		conn := newBenchmarkStreamConn(nil)
		next := 0
		benchmarkRunLoop(b, benchmarkAverageBytes(writePayloads...), nil, func() error {
			payload := writePayloads[next]
			next++
			if next == len(writePayloads) {
				next = 0
			}
			_, err := conn.Write(payload)
			return err
		})
	})
}

func benchmarkTCPRawConnFull(b *testing.B) {
	readCases := []struct {
		name    string
		payload []byte
	}{
		{name: "Read/initiation", payload: benchmarkPayloads.initiation},
		{name: "Read/transport_small", payload: benchmarkPayloads.transportSmall},
	}
	for _, tc := range readCases {
		b.Run(tc.name, func(b *testing.B) {
			conn := newBenchmarkStreamConn(tc.payload)
			buf := make([]byte, benchmarkMaxPacketSize)
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, err := conn.Read(buf)
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
			conn := newBenchmarkStreamConn(nil)
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, err := conn.Write(tc.payload)
				return err
			})
		})
	}
}

func BenchmarkTCPRecordMasquerade(b *testing.B) {
	benchmarkUseDeterministicRand(b)
	if benchmarkFullMatrixEnabled() {
		benchmarkTCPRecordMasqueradeFull(b)
		return
	}

	readPayloads := [][]byte{
		benchmarkPayloads.initiation,
		benchmarkPayloads.transportSmall,
	}
	readEncoded := make([][]byte, len(readPayloads))
	for i, payload := range readPayloads {
		readEncoded[i] = benchmarkEncodeMasqueradeRecord(benchmarkMasqueradeRules, payload)
	}
	b.Run("ReadRecord/mixed", func(b *testing.B) {
		source := newBenchmarkStreamChunksConn(readEncoded)
		pool := benchmarkNewBufferPool()
		conn, ok := NewMasqueradeConn(source, pool, MasqueradeOpts{
			RulesIn:  benchmarkMasqueradeRules,
			RulesOut: benchmarkMasqueradeRules,
		})
		if !ok {
			b.Fatal("expected stream masquerade benchmark conn")
		}
		buf := make([]byte, benchmarkMaxPacketSize)
		benchmarkRunLoop(b, benchmarkAverageBytes(readPayloads...), nil, func() error {
			_, err := conn.ReadRecord(buf)
			return err
		})
	})

	writePayloads := [][]byte{
		benchmarkPayloads.initiation,
		benchmarkPayloads.transportSmall,
		benchmarkPayloads.transportMTU,
	}
	b.Run("WriteRecord/mixed", func(b *testing.B) {
		sink := newBenchmarkStreamConn(nil)
		pool := benchmarkNewBufferPool()
		conn, ok := NewMasqueradeConn(sink, pool, MasqueradeOpts{
			RulesIn:  benchmarkMasqueradeRules,
			RulesOut: benchmarkMasqueradeRules,
		})
		if !ok {
			b.Fatal("expected stream masquerade benchmark conn")
		}
		next := 0
		benchmarkRunLoop(b, benchmarkAverageBytes(writePayloads...), nil, func() error {
			payload := writePayloads[next]
			next++
			if next == len(writePayloads) {
				next = 0
			}
			_, err := conn.WriteRecord(payload)
			return err
		})
	})
}

func benchmarkTCPRecordMasqueradeFull(b *testing.B) {
	readCases := []struct {
		name    string
		payload []byte
	}{
		{name: "ReadRecord/initiation", payload: benchmarkPayloads.initiation},
		{name: "ReadRecord/transport_small", payload: benchmarkPayloads.transportSmall},
	}
	for _, tc := range readCases {
		b.Run(tc.name, func(b *testing.B) {
			source := newBenchmarkStreamConn(benchmarkEncodeStreamRecords(benchmarkMasqueradeRules, tc.payload))
			pool := benchmarkNewBufferPool()
			conn, ok := NewMasqueradeConn(source, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected stream masquerade benchmark conn")
			}
			buf := make([]byte, benchmarkMaxPacketSize)
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, err := conn.ReadRecord(buf)
				return err
			})
		})
	}

	writeCases := []struct {
		name    string
		payload []byte
	}{
		{name: "WriteRecord/initiation", payload: benchmarkPayloads.initiation},
		{name: "WriteRecord/transport_small", payload: benchmarkPayloads.transportSmall},
		{name: "WriteRecord/transport_mtu", payload: benchmarkPayloads.transportMTU},
	}
	for _, tc := range writeCases {
		b.Run(tc.name, func(b *testing.B) {
			sink := newBenchmarkStreamConn(nil)
			pool := benchmarkNewBufferPool()
			conn, ok := NewMasqueradeConn(sink, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected stream masquerade benchmark conn")
			}
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, err := conn.WriteRecord(tc.payload)
				return err
			})
		})
	}
}

func BenchmarkTCPFramed(b *testing.B) {
	benchmarkUseDeterministicRand(b)
	if benchmarkFullMatrixEnabled() {
		benchmarkTCPFramedFull(b)
		return
	}

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
		source := newBenchmarkStreamChunksConn(compatOffEncoded)
		pool := benchmarkNewBufferPool()
		conn, ok := NewFramedConn(source, pool, benchmarkFramedOpts)
		if !ok {
			b.Fatal("expected framed benchmark conn")
		}
		buf := make([]byte, benchmarkMaxPacketSize)
		benchmarkRunLoop(b, benchmarkAverageBytes(compatOffPayloads...), nil, func() error {
			_, err := conn.Read(buf)
			return err
		})
	})
	b.Run("Write/compat_off/mixed", func(b *testing.B) {
		sink := newBenchmarkStreamConn(nil)
		pool := benchmarkNewBufferPool()
		conn, ok := NewFramedConn(sink, pool, benchmarkFramedOpts)
		if !ok {
			b.Fatal("expected framed benchmark conn")
		}
		next := 0
		benchmarkRunLoop(b, benchmarkAverageBytes(compatOffPayloads...), nil, func() error {
			payload := compatOffPayloads[next]
			next++
			if next == len(compatOffPayloads) {
				next = 0
			}
			_, err := conn.Write(payload)
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
		source := newBenchmarkStreamChunksConn(compatOnEncoded)
		pool := benchmarkNewBufferPool()
		conn, ok := NewFramedConn(source, pool, benchmarkFramedCompatOpts)
		if !ok {
			b.Fatal("expected compat framed benchmark conn")
		}
		buf := make([]byte, benchmarkMaxPacketSize)
		benchmarkRunLoop(b, benchmarkAverageBytes(compatOnPayloads...), nil, func() error {
			_, err := conn.Read(buf)
			return err
		})
	})
	b.Run("Write/compat_on/mixed", func(b *testing.B) {
		sink := newBenchmarkStreamConn(nil)
		pool := benchmarkNewBufferPool()
		conn, ok := NewFramedConn(sink, pool, benchmarkFramedCompatOpts)
		if !ok {
			b.Fatal("expected compat framed benchmark conn")
		}
		next := 0
		benchmarkRunLoop(b, benchmarkAverageBytes(compatOnPayloads...), nil, func() error {
			payload := compatOnPayloads[next]
			next++
			if next == len(compatOnPayloads) {
				next = 0
			}
			_, err := conn.Write(payload)
			return err
		})
	})
}

func benchmarkTCPFramedFull(b *testing.B) {
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
			source := newBenchmarkStreamConn(benchmarkEncodeFramedRecord(benchmarkFramedOpts, tc.payload))
			pool := benchmarkNewBufferPool()
			conn, ok := NewFramedConn(source, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected framed benchmark conn")
			}
			buf := make([]byte, benchmarkMaxPacketSize)
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, err := conn.Read(buf)
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
			sink := newBenchmarkStreamConn(nil)
			pool := benchmarkNewBufferPool()
			conn, ok := NewFramedConn(sink, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected framed benchmark conn")
			}
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, err := conn.Write(tc.payload)
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
			source := newBenchmarkStreamConn(benchmarkEncodeFramedRecord(benchmarkFramedCompatOpts, tc.payload))
			pool := benchmarkNewBufferPool()
			conn, ok := NewFramedConn(source, pool, benchmarkFramedCompatOpts)
			if !ok {
				b.Fatal("expected compat framed benchmark conn")
			}
			buf := make([]byte, benchmarkMaxPacketSize)
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, err := conn.Read(buf)
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
			sink := newBenchmarkStreamConn(nil)
			pool := benchmarkNewBufferPool()
			conn, ok := NewFramedConn(sink, pool, benchmarkFramedCompatOpts)
			if !ok {
				b.Fatal("expected compat framed benchmark conn")
			}
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, err := conn.Write(tc.payload)
				return err
			})
		})
	}
}

func BenchmarkTCPPrelude(b *testing.B) {
	benchmarkUseDeterministicRand(b)
	if benchmarkFullMatrixEnabled() {
		benchmarkTCPPreludeFull(b)
		return
	}

	b.Run("ReadCold/mixed_decoys_then_initiation", func(b *testing.B) {
		fixtures := make([]*benchmarkTCPPreludeReadFixture, benchmarkFixtureRingSize)
		for i := range fixtures {
			decoys := [][]byte{{0xaa}}
			if i%2 == 1 {
				decoys = [][]byte{{0xaa}, {0xab}, {0xac}, {0xad}, {0xae}}
			}
			fixtures[i] = newBenchmarkTCPPreludeReadFixture(decoys)
		}
		benchmarkRunLoopWithFixtureRing(b, len(benchmarkPayloads.initiation), fixtures, func(f *benchmarkTCPPreludeReadFixture) {
			f.Reset()
		}, func(f *benchmarkTCPPreludeReadFixture) error {
			return f.Read()
		})
	})

	b.Run("ReadHot/transport_small", func(b *testing.B) {
		pool := benchmarkNewBufferPool()
		source := newBenchmarkStreamConn(benchmarkEncodeStreamRecords(
			benchmarkMasqueradeRules,
			benchmarkEncodeFramedRecord(benchmarkFramedOpts, benchmarkPayloads.transportSmall),
		))
		recordConn, ok := NewMasqueradeConn(source, pool, MasqueradeOpts{
			RulesIn:  benchmarkMasqueradeRules,
			RulesOut: benchmarkMasqueradeRules,
		})
		if !ok {
			b.Fatal("expected record benchmark conn")
		}
		prelude, ok := NewPreludeConn(recordConn, pool, benchmarkFramedOpts, benchmarkPreludeOneRule)
		if !ok {
			b.Fatal("expected prelude benchmark conn")
		}
		prelude.seenValid = true
		buf := make([]byte, benchmarkMaxPacketSize)
		benchmarkRunLoop(b, len(benchmarkPayloads.transportSmall), nil, func() error {
			_, err := prelude.Read(buf)
			return err
		})
	})

	b.Run("Write/initiation_mixed", func(b *testing.B) {
		fixtures := make([]*benchmarkTCPPreludeWriteFixture, benchmarkFixtureRingSize)
		for i := range fixtures {
			opts := benchmarkPreludeOneRule
			if i%2 == 1 {
				opts = benchmarkPreludeFiveRules
			}
			fixtures[i] = newBenchmarkTCPPreludeWriteFixture(
				opts,
				benchmarkEncodeFramedRecord(benchmarkFramedOpts, benchmarkPayloads.initiation),
			)
		}
		benchmarkRunLoopWithFixtureRing(b, len(benchmarkPayloads.initiation), fixtures, nil, func(f *benchmarkTCPPreludeWriteFixture) error {
			return f.Write()
		})
	})

	b.Run("Write/transport_small/passthrough", func(b *testing.B) {
		sink := newBenchmarkStreamConn(nil)
		pool := benchmarkNewBufferPool()
		recordConn, ok := NewMasqueradeConn(sink, pool, MasqueradeOpts{
			RulesIn:  benchmarkMasqueradeRules,
			RulesOut: benchmarkMasqueradeRules,
		})
		if !ok {
			b.Fatal("expected record benchmark conn")
		}
		prelude, ok := NewPreludeConn(recordConn, pool, benchmarkFramedOpts, benchmarkPreludeOneRule)
		if !ok {
			b.Fatal("expected prelude benchmark conn")
		}
		payload := benchmarkEncodeFramedRecord(benchmarkFramedOpts, benchmarkPayloads.transportSmall)
		benchmarkRunLoop(b, len(payload), nil, func() error {
			_, err := prelude.Write(payload)
			return err
		})
	})
}

func benchmarkTCPPreludeFull(b *testing.B) {
	b.Run("ReadCold/decoy1_then_initiation", func(b *testing.B) {
		pool := benchmarkNewBufferPool()
		source := newBenchmarkStreamConn(benchmarkEncodeStreamRecords(
			benchmarkMasqueradeRules,
			[]byte{0xaa},
			benchmarkEncodeFramedRecord(benchmarkFramedOpts, benchmarkPayloads.initiation),
		))
		recordConn, ok := NewMasqueradeConn(source, pool, MasqueradeOpts{
			RulesIn:  benchmarkMasqueradeRules,
			RulesOut: benchmarkMasqueradeRules,
		})
		if !ok {
			b.Fatal("expected record benchmark conn")
		}
		prelude, ok := NewPreludeConn(recordConn, pool, benchmarkFramedOpts, benchmarkPreludeOneRule)
		if !ok {
			b.Fatal("expected prelude benchmark conn")
		}
		buf := make([]byte, benchmarkMaxPacketSize)
		reset := func() {
			source.ResetRead()
			prelude.seenValid = false
		}
		benchmarkRunLoop(b, len(benchmarkPayloads.initiation), reset, func() error {
			_, err := prelude.Read(buf)
			return err
		})
	})

	b.Run("ReadCold/decoy5_then_initiation", func(b *testing.B) {
		pool := benchmarkNewBufferPool()
		source := newBenchmarkStreamConn(benchmarkEncodeStreamRecords(
			benchmarkMasqueradeRules,
			[]byte{0xaa},
			[]byte{0xab},
			[]byte{0xac},
			[]byte{0xad},
			[]byte{0xae},
			benchmarkEncodeFramedRecord(benchmarkFramedOpts, benchmarkPayloads.initiation),
		))
		recordConn, ok := NewMasqueradeConn(source, pool, MasqueradeOpts{
			RulesIn:  benchmarkMasqueradeRules,
			RulesOut: benchmarkMasqueradeRules,
		})
		if !ok {
			b.Fatal("expected record benchmark conn")
		}
		prelude, ok := NewPreludeConn(recordConn, pool, benchmarkFramedOpts, benchmarkPreludeOneRule)
		if !ok {
			b.Fatal("expected prelude benchmark conn")
		}
		buf := make([]byte, benchmarkMaxPacketSize)
		reset := func() {
			source.ResetRead()
			prelude.seenValid = false
		}
		benchmarkRunLoop(b, len(benchmarkPayloads.initiation), reset, func() error {
			_, err := prelude.Read(buf)
			return err
		})
	})

	b.Run("ReadHot/transport_small", func(b *testing.B) {
		pool := benchmarkNewBufferPool()
		source := newBenchmarkStreamConn(benchmarkEncodeStreamRecords(
			benchmarkMasqueradeRules,
			benchmarkEncodeFramedRecord(benchmarkFramedOpts, benchmarkPayloads.transportSmall),
		))
		recordConn, ok := NewMasqueradeConn(source, pool, MasqueradeOpts{
			RulesIn:  benchmarkMasqueradeRules,
			RulesOut: benchmarkMasqueradeRules,
		})
		if !ok {
			b.Fatal("expected record benchmark conn")
		}
		prelude, ok := NewPreludeConn(recordConn, pool, benchmarkFramedOpts, benchmarkPreludeOneRule)
		if !ok {
			b.Fatal("expected prelude benchmark conn")
		}
		prelude.seenValid = true
		buf := make([]byte, benchmarkMaxPacketSize)
		benchmarkRunLoop(b, len(benchmarkPayloads.transportSmall), nil, func() error {
			_, err := prelude.Read(buf)
			return err
		})
	})

	writeCases := []struct {
		name    string
		payload []byte
		opts    PreludeOpts
	}{
		{
			name:    "Write/initiation/1_rule",
			payload: benchmarkEncodeFramedRecord(benchmarkFramedOpts, benchmarkPayloads.initiation),
			opts:    benchmarkPreludeOneRule,
		},
		{
			name:    "Write/initiation/5_rules",
			payload: benchmarkEncodeFramedRecord(benchmarkFramedOpts, benchmarkPayloads.initiation),
			opts:    benchmarkPreludeFiveRules,
		},
		{
			name:    "Write/transport_small/passthrough",
			payload: benchmarkEncodeFramedRecord(benchmarkFramedOpts, benchmarkPayloads.transportSmall),
			opts:    benchmarkPreludeOneRule,
		},
	}
	for _, tc := range writeCases {
		b.Run(tc.name, func(b *testing.B) {
			sink := newBenchmarkStreamConn(nil)
			pool := benchmarkNewBufferPool()
			recordConn, ok := NewMasqueradeConn(sink, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected record benchmark conn")
			}
			prelude, ok := NewPreludeConn(recordConn, pool, benchmarkFramedOpts, tc.opts)
			if !ok {
				b.Fatal("expected prelude benchmark conn")
			}
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, err := prelude.Write(tc.payload)
				return err
			})
		})
	}
}

func BenchmarkTCPPipeline(b *testing.B) {
	benchmarkUseDeterministicRand(b)
	if benchmarkFullMatrixEnabled() {
		benchmarkTCPPipelineFull(b)
		return
	}

	b.Run("Read/full_pipeline/cold_initiation", func(b *testing.B) {
		fixtures := make([]*benchmarkTCPPipelineReadFixture, benchmarkFixtureRingSize)
		for i := range fixtures {
			fixtures[i] = newBenchmarkTCPPipelineReadFixture([][]byte{{0xaa, 0xbb}})
		}
		benchmarkRunLoopWithFixtureRing(b, len(benchmarkPayloads.initiation), fixtures, func(f *benchmarkTCPPipelineReadFixture) {
			f.Reset()
		}, func(f *benchmarkTCPPipelineReadFixture) error {
			return f.Read()
		})
	})

	b.Run("Read/full_pipeline/hot_transport_small", func(b *testing.B) {
		pool := benchmarkNewBufferPool()
		source := newBenchmarkStreamConn(benchmarkEncodeStreamRecords(
			benchmarkMasqueradeRules,
			benchmarkEncodeFramedRecord(benchmarkFramedOpts, benchmarkPayloads.transportSmall),
		))
		recordConn, ok := NewMasqueradeConn(source, pool, MasqueradeOpts{
			RulesIn:  benchmarkMasqueradeRules,
			RulesOut: benchmarkMasqueradeRules,
		})
		if !ok {
			b.Fatal("expected record benchmark conn")
		}
		prelude, ok := NewPreludeConn(recordConn, pool, benchmarkFramedOpts, benchmarkPreludeOneRule)
		if !ok {
			b.Fatal("expected prelude benchmark conn")
		}
		prelude.seenValid = true
		conn, ok := NewFramedConn(prelude, pool, benchmarkFramedOpts)
		if !ok {
			b.Fatal("expected pipeline benchmark conn")
		}
		buf := make([]byte, benchmarkMaxPacketSize)
		benchmarkRunLoop(b, len(benchmarkPayloads.transportSmall), nil, func() error {
			_, err := conn.Read(buf)
			return err
		})
	})

	writePayloads := [][]byte{
		benchmarkPayloads.initiation,
		benchmarkPayloads.transportSmall,
		benchmarkPayloads.transportMTU,
	}
	b.Run("Write/full_pipeline/mixed", func(b *testing.B) {
		sink := newBenchmarkStreamConn(nil)
		pool := benchmarkNewBufferPool()
		recordConn, ok := NewMasqueradeConn(sink, pool, MasqueradeOpts{
			RulesIn:  benchmarkMasqueradeRules,
			RulesOut: benchmarkMasqueradeRules,
		})
		if !ok {
			b.Fatal("expected record benchmark conn")
		}
		prelude, ok := NewPreludeConn(recordConn, pool, benchmarkFramedOpts, benchmarkPreludeOneRule)
		if !ok {
			b.Fatal("expected prelude benchmark conn")
		}
		conn, ok := NewFramedConn(prelude, pool, benchmarkFramedOpts)
		if !ok {
			b.Fatal("expected pipeline benchmark conn")
		}
		next := 0
		benchmarkRunLoop(b, benchmarkAverageBytes(writePayloads...), nil, func() error {
			payload := writePayloads[next]
			next++
			if next == len(writePayloads) {
				next = 0
			}
			_, err := conn.Write(payload)
			return err
		})
	})
}

func benchmarkTCPPipelineFull(b *testing.B) {
	b.Run("Read/full_pipeline/cold_initiation", func(b *testing.B) {
		pool := benchmarkNewBufferPool()
		source := newBenchmarkStreamConn(benchmarkEncodeStreamRecords(
			benchmarkMasqueradeRules,
			[]byte{0xaa, 0xbb},
			benchmarkEncodeFramedRecord(benchmarkFramedOpts, benchmarkPayloads.initiation),
		))
		recordConn, ok := NewMasqueradeConn(source, pool, MasqueradeOpts{
			RulesIn:  benchmarkMasqueradeRules,
			RulesOut: benchmarkMasqueradeRules,
		})
		if !ok {
			b.Fatal("expected record benchmark conn")
		}
		prelude, ok := NewPreludeConn(recordConn, pool, benchmarkFramedOpts, benchmarkPreludeOneRule)
		if !ok {
			b.Fatal("expected prelude benchmark conn")
		}
		conn, ok := NewFramedConn(prelude, pool, benchmarkFramedOpts)
		if !ok {
			b.Fatal("expected pipeline benchmark conn")
		}
		buf := make([]byte, benchmarkMaxPacketSize)
		reset := func() {
			source.ResetRead()
			prelude.seenValid = false
		}
		benchmarkRunLoop(b, len(benchmarkPayloads.initiation), reset, func() error {
			_, err := conn.Read(buf)
			return err
		})
	})

	b.Run("Read/full_pipeline/hot_transport_small", func(b *testing.B) {
		pool := benchmarkNewBufferPool()
		source := newBenchmarkStreamConn(benchmarkEncodeStreamRecords(
			benchmarkMasqueradeRules,
			benchmarkEncodeFramedRecord(benchmarkFramedOpts, benchmarkPayloads.transportSmall),
		))
		recordConn, ok := NewMasqueradeConn(source, pool, MasqueradeOpts{
			RulesIn:  benchmarkMasqueradeRules,
			RulesOut: benchmarkMasqueradeRules,
		})
		if !ok {
			b.Fatal("expected record benchmark conn")
		}
		prelude, ok := NewPreludeConn(recordConn, pool, benchmarkFramedOpts, benchmarkPreludeOneRule)
		if !ok {
			b.Fatal("expected prelude benchmark conn")
		}
		prelude.seenValid = true
		conn, ok := NewFramedConn(prelude, pool, benchmarkFramedOpts)
		if !ok {
			b.Fatal("expected pipeline benchmark conn")
		}
		buf := make([]byte, benchmarkMaxPacketSize)
		benchmarkRunLoop(b, len(benchmarkPayloads.transportSmall), nil, func() error {
			_, err := conn.Read(buf)
			return err
		})
	})

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
			sink := newBenchmarkStreamConn(nil)
			pool := benchmarkNewBufferPool()
			recordConn, ok := NewMasqueradeConn(sink, pool, MasqueradeOpts{
				RulesIn:  benchmarkMasqueradeRules,
				RulesOut: benchmarkMasqueradeRules,
			})
			if !ok {
				b.Fatal("expected record benchmark conn")
			}
			prelude, ok := NewPreludeConn(recordConn, pool, benchmarkFramedOpts, benchmarkPreludeOneRule)
			if !ok {
				b.Fatal("expected prelude benchmark conn")
			}
			conn, ok := NewFramedConn(prelude, pool, benchmarkFramedOpts)
			if !ok {
				b.Fatal("expected pipeline benchmark conn")
			}
			benchmarkRunLoop(b, len(tc.payload), nil, func() error {
				_, err := conn.Write(tc.payload)
				return err
			})
		})
	}
}

type benchmarkTCPPreludeReadFixture struct {
	source  *benchmarkStreamConn
	prelude *PreludeConn
	buf     []byte
}

func newBenchmarkTCPPreludeReadFixture(decoys [][]byte) *benchmarkTCPPreludeReadFixture {
	pool := benchmarkNewBufferPool()
	records := make([][]byte, 0, len(decoys)+1)
	records = append(records, decoys...)
	records = append(records, benchmarkEncodeFramedRecord(benchmarkFramedOpts, benchmarkPayloads.initiation))
	source := newBenchmarkStreamConn(benchmarkEncodeStreamRecords(benchmarkMasqueradeRules, records...))
	recordConn, ok := NewMasqueradeConn(source, pool, MasqueradeOpts{
		RulesIn:  benchmarkMasqueradeRules,
		RulesOut: benchmarkMasqueradeRules,
	})
	if !ok {
		panic("expected record benchmark conn")
	}
	prelude, ok := NewPreludeConn(recordConn, pool, benchmarkFramedOpts, benchmarkPreludeOneRule)
	if !ok {
		panic("expected prelude benchmark conn")
	}
	return &benchmarkTCPPreludeReadFixture{
		source:  source,
		prelude: prelude,
		buf:     make([]byte, benchmarkMaxPacketSize),
	}
}

func (f *benchmarkTCPPreludeReadFixture) Reset() {
	f.source.ResetRead()
	f.prelude.seenValid = false
}

func (f *benchmarkTCPPreludeReadFixture) Read() error {
	_, err := f.prelude.Read(f.buf)
	return err
}

type benchmarkTCPPreludeWriteFixture struct {
	prelude *PreludeConn
	payload []byte
}

func newBenchmarkTCPPreludeWriteFixture(opts PreludeOpts, payload []byte) *benchmarkTCPPreludeWriteFixture {
	sink := newBenchmarkStreamConn(nil)
	pool := benchmarkNewBufferPool()
	recordConn, ok := NewMasqueradeConn(sink, pool, MasqueradeOpts{
		RulesIn:  benchmarkMasqueradeRules,
		RulesOut: benchmarkMasqueradeRules,
	})
	if !ok {
		panic("expected record benchmark conn")
	}
	prelude, ok := NewPreludeConn(recordConn, pool, benchmarkFramedOpts, opts)
	if !ok {
		panic("expected prelude benchmark conn")
	}
	return &benchmarkTCPPreludeWriteFixture{
		prelude: prelude,
		payload: payload,
	}
}

func (f *benchmarkTCPPreludeWriteFixture) Write() error {
	_, err := f.prelude.Write(f.payload)
	return err
}

type benchmarkTCPPipelineReadFixture struct {
	source  *benchmarkStreamConn
	prelude *PreludeConn
	conn    *FramedConn
	buf     []byte
}

func newBenchmarkTCPPipelineReadFixture(decoys [][]byte) *benchmarkTCPPipelineReadFixture {
	pool := benchmarkNewBufferPool()
	records := make([][]byte, 0, len(decoys)+1)
	records = append(records, decoys...)
	records = append(records, benchmarkEncodeFramedRecord(benchmarkFramedOpts, benchmarkPayloads.initiation))
	source := newBenchmarkStreamConn(benchmarkEncodeStreamRecords(benchmarkMasqueradeRules, records...))
	recordConn, ok := NewMasqueradeConn(source, pool, MasqueradeOpts{
		RulesIn:  benchmarkMasqueradeRules,
		RulesOut: benchmarkMasqueradeRules,
	})
	if !ok {
		panic("expected record benchmark conn")
	}
	prelude, ok := NewPreludeConn(recordConn, pool, benchmarkFramedOpts, benchmarkPreludeOneRule)
	if !ok {
		panic("expected prelude benchmark conn")
	}
	conn, ok := NewFramedConn(prelude, pool, benchmarkFramedOpts)
	if !ok {
		panic("expected pipeline benchmark conn")
	}
	return &benchmarkTCPPipelineReadFixture{
		source:  source,
		prelude: prelude,
		conn:    conn,
		buf:     make([]byte, benchmarkMaxPacketSize),
	}
}

func (f *benchmarkTCPPipelineReadFixture) Reset() {
	f.source.ResetRead()
	f.prelude.seenValid = false
}

func (f *benchmarkTCPPipelineReadFixture) Read() error {
	_, err := f.conn.Read(f.buf)
	return err
}
