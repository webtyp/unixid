---
PLAN: "feat!: explicit replica ids — NewForReplica(replica, last), monotonic under clock regression, typed constructor"
TAG: v0.3.0
EXECUTOR: jules
REVIEWER: none
---

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.

# Plan — `unixid`: explicit replica, strictly monotonic ids, typed constructor

## 0. Context (read first)

`webtyp.com/unixid` mints primary keys as text: `<unix nanoseconds>` on a server
(`"1624397134562544800"`) and `<unix nanoseconds>.<n>` elsewhere. An offline-first wave
(master plan, in Spanish: <https://github.com/veltylabs/mjosefa-cms/blob/main/docs/OFFLINE_FIRST_MASTER_PLAN.md>)
makes every browser a **replica** that creates records while the server is unreachable and
syncs them later. Ids minted on two devices must never collide, and an id minted on one
device must never repeat, or a later sync treats a new record as a replay of an old one and
drops it.

Defects in the current code (verified on `main`, v0.2.29):

1. `unixIdNano()` only compares `currentUnixNano == id.lastUnixNano`. If the wall clock moves
   **backwards** (an NTP adjustment), it returns a timestamp it already returned: a duplicate id.
2. `NewUnixID(...any)` with a `*sync.Mutex` argument installs `defaultNoOpMutex`, so the
   generator **stops locking**: a data race under concurrent callers.
3. The suffix comes from `userSessionNumber`, an interface with an **unexported** method, so no
   package outside `unixid` can implement it: the suffix is unreachable in practice. In the
   browser every id is a bare timestamp, and two devices can mint the same one.
4. The public constructor takes `...any` and silently ignores arguments of the wrong type.
5. Two build-tagged files (`unixid_front.go` `wasm`, `unixid_back.go` `!wasm`) behave differently
   for the same call.

## Design gate

### 1. Prior art
- **Twitter Snowflake / Discord / Instagram ids**: timestamp + **explicit worker/node id**
  assigned by a coordinator + per-node sequence. Collision-free by construction because the
  node id is unique; monotonic per node. This plan is that family, keeping the existing
  decimal-text layout.
- **ULID / UUIDv7 (RFC 9562, 2024)**: timestamp + random bits; the spec's "monotonic" mode
  increments the previous value when the clock does not advance or goes back. We copy that
  rule (`next = max(now, last+1)`), but not the random part: ids already in production (legacy
  import keeps the decimal nanosecond text) must keep one format, and TinyGo binaries avoid a
  CSPRNG dependency.
- **MongoDB ObjectId**: timestamp + machine/process id + counter; same "who minted it is part
  of the id" principle.

Ours differs in that the replica number is **assigned by the server** (a later `dbsync`
library) and passed in explicitly; this library never invents one.

### 2. Novice-name test
- `unixid.NewUnixID()` — "a new unix-id generator" (server; ids without suffix). Name kept.
- `unixid.NewForReplica(replica, last)` — "a new generator for replica 7, continuing after the
  last id it minted". `Replica` is a named type so a bare int cannot be passed by accident.
- `Parse(id) (timestamp, replica, err)` — returns a `Replica` (0 = minted by a server) instead
  of an untyped `userNum string`.

### 3. Complexity ledger
```
Concepts the developer must learn   +1 (Replica) / −2 (session-number handler, external mutex)
Files they must touch to do X        0
Lines at the call site               0 for the 45 existing `unixid.NewUnixID()` calls (unchanged)
Ways to do the same thing            −1 (the ...any constructor with type-switched options dies)
```

### 4. Where it belongs
Id minting is this library's single concern. Assigning replica numbers belongs to the sync
library (`webtyp/dbsync`, separate plan); persisting the last minted id belongs to its caller.

### 5. What this deletes
`Config`, `lockHandler`, `userSessionNumber`, `defaultEmptySession`, `defaultNoOpMutex`,
`createUnixID`, the files `unixid_front.go` and `unixid_back.go`, the `correlativeNumber` and
`userNum` fields, and the `...any` parameter of `NewUnixID`.

## 1. Target API (exactly this, nothing more exported)

```go
package unixid

// Replica identifies one device that mints ids while offline. The server assigns it
// (never this library). 0 is not a valid replica: it means "minted by a server".
type Replica uint32

// ErrReplicaZero is returned by NewForReplica when replica == 0.
// (declare as a typed constant of an unexported error type, see §2)

// NewUnixID returns the server generator: ids are "<nanoseconds>" with no suffix.
// The error is always nil today; it is kept so both constructors share one shape.
func NewUnixID() (*UnixID, error)

// NewForReplica returns a generator for one device: ids are "<nanoseconds>.<replica>".
// last is the timestamp part of the newest id this replica already minted (0 on a fresh
// device); every id this generator returns has a timestamp strictly greater than last.
func NewForReplica(replica Replica, last int64) (*UnixID, error)

func (u *UnixID) NewID() string            // satisfies webtyp.com/model.IDGenerator
func (u *UnixID) SetNewID(target *string)  // unchanged
func (u *UnixID) Validate(id string) error // unchanged rules
func (u *UnixID) Parse(id string) (timestamp int64, replica Replica, err error)
```

## 2. Stages

### Stage 1 — rewrite `unixid.go`; delete the build-tagged files
- Delete `unixid_front.go` and `unixid_back.go`.
- `UnixID` struct holds exactly: `mu sync.Mutex`, `last int64`, `suffix string` (`""` for the
  server, `"." + decimal(replica)` for a replica, computed once in the constructor with
  `webtyp.com/fmt` `Convert(...)`, never per call), `buf []byte` if still used.
- `sync.Mutex` from the standard library is used in **both** targets (TinyGo supports it). Do
  not add build tags.
- Minting rule, under the mutex:
  ```go
  now := time.Now()           // webtyp.com/time, nanoseconds
  if now <= u.last { now = u.last + 1 }
  u.last = now
  return <decimal(now)> + u.suffix
  ```
- `NewForReplica`: `replica == 0` → return `nil, ErrReplicaZero`; otherwise `u.last = last`.
- Error constant, following the house pattern (no `errors` package; comparisons by type
  assertion, never `==` between interfaces under TinyGo):
  ```go
  type idError string
  func (e idError) Error() string { return string(e) }
  const ErrReplicaZero idError = "unixid: replica 0 is reserved for servers"
  ```
- Keep a package-level unexported `var now = time.Now` **only if** Stage 3's clock-regression
  test needs it (see §3); the minting rule then calls `now()`.

### Stage 2 — `parse.go`
`Parse` returns `(timestamp int64, replica Replica, err error)`. The suffix after the single
`.` is parsed as a base-10 `uint32`; a suffix that is not a valid `uint32`, or equals `0`,
returns the existing "format invalid" error. No suffix → replica 0. `Validate` is unchanged.
Update every doc comment that says "user number" / "session" to say "replica".

### Stage 3 — tests (`tests/`, external package `unixid_test`)
Update the existing tests to the new API and add:
1. `NewForReplica(0, 0)` → returns a nil generator and an error whose `Error()` equals
   `unixid.ErrReplicaZero.Error()`.
2. Suffix: `NewForReplica(42, 0)` → every id ends in `.42`; `Parse` returns `Replica(42)`.
3. Server: `NewUnixID()` ids contain no `.`; `Parse` returns `Replica(0)`.
4. Monotonic: 100 000 ids from one generator, each strictly greater than the previous when
   compared by the timestamp from `Parse`, and all distinct.
5. Concurrency: 8 goroutines × 10 000 ids, no duplicates (run under `-race` in the stdlib lane).
6. `last` honoured: `NewForReplica(7, far)` where `far = time.Now() + 3600e9` (an hour in the
   future) → the first id's timestamp is `far + 1`.
7. Clock regression: this needs to control the clock. Put it in a **root-level** test file
   `clock_regression_test.go` (package `unixid`) whose first line is
   `// Root-level test (justified): drives the unexported clock to prove ids stay monotonic when the wall clock goes backwards.`
   It swaps the unexported `now` var to return 100, then 50: the second id's timestamp is 101.
   **Never export a symbol for a test.**

Tests run with `gotest` (never plain `go test`), which also runs the wasm lane.

### Stage 4 — docs
- `README.md`: replace the "Client-side (WebAssembly) Usage" section (session handler) and the
  "Thread Safety & Avoiding Deadlocks" section (external mutex) with:
  - an "I want X → use Y" table: server ids → `NewUnixID()`; offline device ids →
    `NewForReplica(replica, last)`; read an id back → `Parse`;
  - one paragraph explaining why `last` exists (a device restarted after its clock moved back
    must not repeat an id) and that the replica number is assigned by the server.
  - ID format section: server `1624397134562544800`, replica `1624397134562544800.42`; ids of
    one generator sort lexicographically while timestamps have 19 digits (until year 2286).

## 3. Code rules (non-negotiable)
- `webtyp.com/fmt` for formatting and errors; no `errors`, `strconv`, `strings` from the
  standard library (this package compiles to WASM). `sync` is allowed.
- No string literal repeated in logic: the `"."` separator is an unexported constant.
- No exported symbol beyond §1.

## 4. Acceptance criteria
- `gotest ./...` green (stdlib + wasm).
- `grep -rn "userSessionNumber\|defaultNoOpMutex\|lockHandler\|createUnixID\|Config struct" --include=*.go .` → empty.
- `test ! -e unixid_front.go && test ! -e unixid_back.go`.
- `grep -rn "func NewUnixID(" unixid.go` shows a signature with **no parameters**.
- `grep -rn "\.\.\.any" --include=*.go . | grep -v _test.go` → empty.

| Stage | Files | Done when |
|---|---|---|
| 1 | `unixid.go` (rewrite), delete `unixid_front.go`, `unixid_back.go` | builds in both targets |
| 2 | `parse.go` | `Parse` returns `Replica` |
| 3 | `tests/*.go`, `clock_regression_test.go` | 7 cases green |
| 4 | `README.md` | table + format section updated |
