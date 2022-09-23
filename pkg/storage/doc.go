// Copyright 2014 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

/*
Package storage provides low-level storage. It interacts with storage
backends (e.g. LevelDB, RocksDB, etc.) via the Engine interface. At
one level higher, MVCC provides multi-version concurrency control
capability on top of an Engine instance.

The Engine interface provides an API for key-value stores. InMem
implements an in-memory engine using a sorted map. RocksDB implements
an engine for data stored to local disk using RocksDB, a variant of
LevelDB.

MVCC provides a multi-version concurrency control system on top of an
engine. MVCC is the basis for Cockroach's support for distributed
transactions. It is intended for direct use from storage.Range
objects.

# Notes on MVCC architecture

Each MVCC value contains a metadata key/value pair and one or more
version key/value pairs. The MVCC metadata key is the actual key for
the value, using the util/encoding.EncodeBytes scheme. The MVCC
metadata value is of type MVCCMetadata and contains the most recent
version timestamp and an optional roachpb.Transaction message. If
set, the most recent version of the MVCC value is a transactional
"intent". It also contains some information on the size of the most
recent version's key and value for efficient stat counter
computations. Note that it is not necessary to explicitly store the
MVCC metadata as its contents can be reconstructed from the most
recent versioned value as long as an intent is not present. The
implementation takes advantage of this and deletes the MVCC metadata
when possible.

Each MVCC version key/value pair has a key which is also
binary-encoded, but is suffixed with a decreasing, big-endian encoding
of the timestamp (eight bytes for the nanosecond wall time, followed
by four bytes for the logical time except for meta key value pairs,
for which the timestamp is implicit). The MVCC version value is
a message of type roachpb.Value. A deletion is indicated by an
empty value. Note that an empty roachpb.Value will encode to
a non-empty byte slice. The decreasing encoding on the timestamp sorts
the most recent version directly after the metadata key, which is
treated specially by the RocksDB comparator (by making the zero
timestamp sort first). This increases the likelihood that an
Engine.Get() of the MVCC metadata will get the same block containing
the most recent version, even if there are many versions. We rely on
getting the MVCC metadata key/value and then using it to directly get
the MVCC version using the metadata's most recent version timestamp.
This avoids using an expensive merge iterator to scan the most recent
version. It also allows us to leverage RocksDB's bloom filters.

The following is an example of the sort order for MVCC key/value pairs:

	...
	keyA: MVCCMetadata of keyA
	keyA_Timestamp_n: value of version_n
	keyA_Timestamp_n-1: value of version_n-1
	...
	keyA_Timestamp_0: value of version_0
	keyB: MVCCMetadata of keyB

The binary encoding used on the MVCC keys allows arbitrary keys to be
stored in the map (no restrictions on intermediate nil-bytes, for
example), while still sorting lexicographically and guaranteeing that
all timestamp-suffixed MVCC version keys sort consecutively with the
metadata key. We use an escape-based encoding which transforms all nul
("\x00") characters in the key and is terminated with the sequence
"\x00\x01", which is guaranteed to not occur elsewhere in the encoded
value. See util/encoding/encoding.go for more details.

We considered inlining the most recent MVCC version in the
MVCCMetadata. This would reduce the storage overhead of storing the
same key twice (which is small due to block compression), and the
runtime overhead of two separate DB lookups. On the other hand, all
writes that create a new version of an existing key would incur a
double write as the previous value is moved out of the MVCCMetadata
into its versioned key. Preliminary benchmarks have not shown enough
performance improvement to justify this change, although we may
revisit this decision if it turns out that multiple versions of the
same key are rare in practice.

However, we do allow inlining in order to use the MVCC interface to
store non-versioned values. It turns out that not everything which
Cockroach needs to store would be efficient or possible using MVCC.
Examples include transaction records, abort span entries, stats
counters, time series data, and system-local config values. However,
supporting a mix of encodings is problematic in terms of resulting
complexity. So Cockroach treats an MVCC timestamp of zero to mean an
inlined, non-versioned value. These values are replaced if they exist
on a Put operation and are cleared from the engine on a delete.
Importantly, zero-timestamped MVCC values may be merged, as is
necessary for stats counters and time series data.

# Versioning

CockroachDB uses cluster versions to accommodate backwards-incompatible
behavior. Cluster versions are described and documented in
pkg/clusterversion. The negotiation of cluster versions during cluster
initialization and version upgrades maintains an 'active' cluster
version that all nodes operate at. During an upgrade, this active
cluster version is ratcheted upwards only when all nodes are running
binaries capable of supporting the new active cluster version. Once an
active cluster version is set, none of the cluster's nodes may be
downgraded to a binary that does not understand the new cluster version.

When a new active cluster version has been negotiated, during cluster
initialization or a version upgrade, the new active cluster version is
durably written to every store on every node (see kvserver's
WriteClusterVersionToEngines). If a node has multiple stores, the first
store's cluster version determines the cluster version that the node
operates at. The active cluster version that is persisted to an
individual store is referred to as the "store cluster version". This
cluster version is stored in a store-local key
keys.StoreClusterVersionKey() within the store itself.

Previously, as a consequence of storing the store cluster key within the
store, a store's cluster version could not be known until after the
store was opened. In order to address this limitation, a separate
`STORAGE_MIN_VERSION` file was introduced in v21.2 to store the store
cluster version outside the store itself. This file stores a cluster
version protocol buffer and may be read without opening the store
itself. Whenever a store must persist a new active cluster version, it
writes to the store cluster version key and calls the engine's
SetMinVersion method to update the `STORAGE_MIN_VERSION` file.  The
engine's SetMinVersion invocation also serves as the storage engine's
opportunity to enable its own backwards-incompatible features when
required.

Pebble has its own concept of backwards-incompatibility gating, using
'format major versions.' Format major versions are tied to CockroachDB
cluster versions. When a store's active cluster version is updated and
SetMinVersion is called, there is a guarantee that the store will never
be opened with a binary that does not understand the new cluster
version. If there's a format major version associated with the cluster
version, SetMinVersion ratchets Pebble's format major version, enabling
the new backwards-incompatible format. Since this ratcheting is not
atomic with the persistence of the `STORAGE_MIN_VERSION` file, a crash
before the format major version is ratcheted may leave Pebble using an
old format. To account for this failure scenario, opening a store also
re-sets the the min-version on store open. Setting the min version again
doesn't ratchet the min version, but performs the necessary mapping from
cluster version to format major version and potentially ratcheting the
format major version. Because of this `SetMinVersion` must handle
repeated calls at the same version, and unconditionally map the version
to a Pebble format major version.

Future work may remove the store cluster version key, instead relying on
the `STORAGE_MIN_VERSION` as the single source of truth on a store's
cluster version.
*/
package storage
