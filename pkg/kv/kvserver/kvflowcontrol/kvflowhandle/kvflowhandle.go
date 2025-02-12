// Copyright 2023 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package kvflowhandle

import (
	"context"
	"sort"
	"time"

	"github.com/cockroachdb/cockroach/pkg/kv/kvserver/kvflowcontrol"
	"github.com/cockroachdb/cockroach/pkg/kv/kvserver/kvflowcontrol/kvflowcontrolpb"
	"github.com/cockroachdb/cockroach/pkg/kv/kvserver/kvflowcontrol/kvflowtokentracker"
	"github.com/cockroachdb/cockroach/pkg/util/admission/admissionpb"
	"github.com/cockroachdb/cockroach/pkg/util/hlc"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/syncutil"
)

// Handle is a concrete implementation of the kvflowcontrol.Handle
// interface. It's held on replicas initiating replication traffic, managing
// multiple Streams (one per active replica) underneath.
type Handle struct {
	controller kvflowcontrol.Controller
	metrics    *Metrics
	clock      *hlc.Clock

	mu struct {
		syncutil.Mutex
		connections []*connectedStream
		// perStreamTokenTracker tracks flow token deductions for each stream.
		// It's used to release tokens back to the controller once log entries
		// (identified by their log positions) have been admitted below-raft,
		// streams disconnect, or the handle closed entirely.
		perStreamTokenTracker map[kvflowcontrol.Stream]*kvflowtokentracker.Tracker
		closed                bool
	}
}

// New constructs a new Handle.
func New(controller kvflowcontrol.Controller, metrics *Metrics, clock *hlc.Clock) *Handle {
	h := &Handle{
		controller: controller,
		metrics:    metrics,
		clock:      clock,
	}
	h.mu.perStreamTokenTracker = map[kvflowcontrol.Stream]*kvflowtokentracker.Tracker{}
	return h
}

var _ kvflowcontrol.Handle = &Handle{}

// Admit is part of the kvflowcontrol.Handle interface.
func (h *Handle) Admit(ctx context.Context, pri admissionpb.WorkPriority, ct time.Time) error {
	if h == nil {
		// TODO(irfansharif): This can happen if we're proposing immediately on
		// a newly split off RHS that doesn't know it's a leader yet (so we
		// haven't initialized a handle). We don't want to deduct/track flow
		// tokens for it; the handle only has a lifetime while we explicitly
		// know that we're the leaseholder+leader. It's ok for the caller to
		// later invoke ReturnTokensUpto even with a no-op DeductTokensFor since
		// it can only return what has been actually been deducted.
		//
		// As for cluster settings that disable flow control entirely or only
		// for regular traffic, that can be dealt with at the caller by not
		// calling .Admit() and ensuring we use the right raft entry encodings.
		return nil
	}

	h.mu.Lock()
	if h.mu.closed {
		log.Errorf(ctx, "operating on a closed handle")
		return nil
	}
	connections := h.mu.connections
	h.mu.Unlock()

	class := admissionpb.WorkClassFromPri(pri)
	h.metrics.onWaiting(class)
	tstart := h.clock.PhysicalTime()

	for _, c := range connections {
		if err := h.controller.Admit(ctx, pri, ct, c); err != nil {
			h.metrics.onErrored(class, h.clock.PhysicalTime().Sub(tstart))
			return err
		}
	}

	h.metrics.onAdmitted(class, h.clock.PhysicalTime().Sub(tstart))
	return nil
}

// DeductTokensFor is part of the kvflowcontrol.Handle interface.
func (h *Handle) DeductTokensFor(
	ctx context.Context,
	pri admissionpb.WorkPriority,
	pos kvflowcontrolpb.RaftLogPosition,
	tokens kvflowcontrol.Tokens,
) {
	if h == nil {
		// TODO(irfansharif): See TODO around nil receiver check in Admit().
		return
	}

	_ = h.deductTokensForInner(ctx, pri, pos, tokens)
}

func (h *Handle) deductTokensForInner(
	ctx context.Context,
	pri admissionpb.WorkPriority,
	pos kvflowcontrolpb.RaftLogPosition,
	tokens kvflowcontrol.Tokens,
) (streams []kvflowcontrol.Stream) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.mu.closed {
		log.Errorf(ctx, "operating on a closed handle")
		return nil // unused return value in production code
	}

	for _, c := range h.mu.connections {
		h.controller.DeductTokens(ctx, pri, tokens, c.Stream())
		h.mu.perStreamTokenTracker[c.Stream()].Track(ctx, pri, tokens, pos)
		streams = append(streams, c.Stream())
	}
	return streams
}

// ReturnTokensUpto is part of the kvflowcontrol.Handle interface.
func (h *Handle) ReturnTokensUpto(
	ctx context.Context,
	pri admissionpb.WorkPriority,
	upto kvflowcontrolpb.RaftLogPosition,
	stream kvflowcontrol.Stream,
) {
	if h == nil {
		// We're trying to release tokens to a handle that no longer exists,
		// likely because we've lost the lease and/or raft leadership since
		// we acquired flow tokens originally. At that point the handle was
		// closed, and all flow tokens were returned back to the controller.
		// There's nothing left for us to do here.
		//
		// NB: It's possible to have reacquired leadership and re-initialize a
		// handle. We still want to ignore token returns from earlier
		// terms/leases (which were already returned to the controller). To that
		// end, we rely on the handle being re-initialized with an empty tracker
		// -- there's simply nothing to double return. Also, when connecting
		// streams on fresh handles, we specify a lower-bound raft log position.
		// The log position corresponds to when the lease/leadership was
		// acquired (whichever comes after). This is used to assert against
		// regressions in token deductions (i.e. deducting tokens for indexes
		// lower than the current term/lease).
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.mu.closed {
		log.Errorf(ctx, "operating on a closed handle")
		return
	}

	tokens := h.mu.perStreamTokenTracker[stream].Untrack(ctx, pri, upto)
	h.controller.ReturnTokens(ctx, pri, tokens, stream)
}

// ConnectStream is part of the kvflowcontrol.Handle interface.
func (h *Handle) ConnectStream(
	ctx context.Context, pos kvflowcontrolpb.RaftLogPosition, stream kvflowcontrol.Stream,
) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.mu.closed {
		log.Errorf(ctx, "operating on a closed handle")
		return
	}

	if _, ok := h.mu.perStreamTokenTracker[stream]; ok {
		log.Fatalf(ctx, "reconnecting already connected stream: %s", stream)
	}
	h.mu.connections = append(h.mu.connections, newConnectedStream(stream))
	sort.Slice(h.mu.connections, func(i, j int) bool {
		// Sort connections based on store IDs (this is the order in which we
		// invoke Controller.Admit) for predictability. If in the future we use
		// flow tokens for raft log catchup (see I11 and [^9] in
		// kvflowcontrol/doc.go), we may want to introduce an Admit-variant that
		// both blocks and deducts tokens before sending catchup MsgApps. In
		// that case, this sorting will help avoid deadlocks.
		return h.mu.connections[i].Stream().StoreID < h.mu.connections[j].Stream().StoreID
	})
	h.mu.perStreamTokenTracker[stream] = kvflowtokentracker.New(pos, nil /* knobs */)
}

// DisconnectStream is part of the kvflowcontrol.Handle interface.
func (h *Handle) DisconnectStream(ctx context.Context, stream kvflowcontrol.Stream) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.disconnectStreamLocked(ctx, stream)
}

func (h *Handle) disconnectStreamLocked(ctx context.Context, stream kvflowcontrol.Stream) {
	if h.mu.closed {
		log.Errorf(ctx, "operating on a closed handle")
		return
	}
	if _, ok := h.mu.perStreamTokenTracker[stream]; !ok {
		log.Fatalf(ctx, "disconnecting non-existent stream: %s", stream)
	}

	h.mu.perStreamTokenTracker[stream].Iter(ctx,
		func(pri admissionpb.WorkPriority, tokens kvflowcontrol.Tokens) {
			h.controller.ReturnTokens(ctx, pri, tokens, stream)
		},
	)
	delete(h.mu.perStreamTokenTracker, stream)

	streamIdx := -1
	for i := range h.mu.connections {
		if h.mu.connections[i].Stream() == stream {
			streamIdx = i
			break
		}
	}
	connection := h.mu.connections[streamIdx]
	connection.Disconnect()
	h.mu.connections = append(h.mu.connections[:streamIdx], h.mu.connections[streamIdx+1:]...)

	// TODO(irfansharif): Optionally record lower bound raft log positions for
	// disconnected streams to guard against regressions when (re-)connecting --
	// it must be done with higher positions.
}

// Close is part of the kvflowcontrol.Handle interface.
func (h *Handle) Close(ctx context.Context) {
	if h == nil {
		return // nothing to do
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.mu.closed {
		log.Errorf(ctx, "operating on a closed handle")
		return
	}

	for _, connection := range h.mu.connections {
		h.disconnectStreamLocked(ctx, connection.Stream())
	}
	h.mu.closed = true
}

// TestingNonBlockingAdmit is a non-blocking alternative to Admit() for use in
// tests.
//   - it checks if we have a non-zero number of flow tokens for all connected
//     streams;
//   - if we do, we return immediately with admitted=true;
//   - if we don't, we return admitted=false and two sets of callbacks:
//     (i) signaled, which can be polled to check whether we're ready to try and
//     admitting again. There's one per underlying stream.
//     (ii) admit, which can be used to try and admit again. If still not
//     admitted, callers are to wait until they're signaled again. There's one
//     per underlying stream.
func (h *Handle) TestingNonBlockingAdmit(
	ctx context.Context, pri admissionpb.WorkPriority,
) (admitted bool, signaled []func() bool, admit []func() bool) {
	h.mu.Lock()
	if h.mu.closed {
		log.Fatalf(ctx, "operating on a closed handle")
	}
	connections := h.mu.connections
	h.mu.Unlock()

	type testingNonBlockingController interface {
		TestingNonBlockingAdmit(
			pri admissionpb.WorkPriority, connection kvflowcontrol.ConnectedStream,
		) (admitted bool, signaled func() bool, admit func() bool)
	}

	tstart := h.clock.PhysicalTime()
	class := admissionpb.WorkClassFromPri(pri)
	h.metrics.onWaiting(class)

	admitted = true
	controller := h.controller.(testingNonBlockingController)
	for _, c := range connections {
		connectionAdmitted, connectionSignaled, connectionAdmit := controller.TestingNonBlockingAdmit(pri, c)
		if connectionAdmitted {
			continue
		}

		admit = append(admit, func() bool {
			if connectionAdmit() {
				h.metrics.onAdmitted(class, h.clock.PhysicalTime().Sub(tstart))
				return true
			}
			return false
		})
		signaled = append(signaled, connectionSignaled)
		admitted = false
	}
	if admitted {
		h.metrics.onAdmitted(class, h.clock.PhysicalTime().Sub(tstart))
	}
	return admitted, signaled, admit
}

// TestingDeductTokensForInner exposes deductTokensForInner for testing
// purposes.
func (h *Handle) TestingDeductTokensForInner(
	ctx context.Context,
	pri admissionpb.WorkPriority,
	pos kvflowcontrolpb.RaftLogPosition,
	tokens kvflowcontrol.Tokens,
) []kvflowcontrol.Stream {
	return h.deductTokensForInner(ctx, pri, pos, tokens)
}
