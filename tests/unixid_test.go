package unixid__test

import (
	"sync"
	"testing"
	"time"
	. "webtyp.com/unixid"
)

// Test_GetNewID prueba el flujo completo de generación de IDs
func Test_GetNewID(t *testing.T) {
	idRequired := 10000
	wg := sync.WaitGroup{}
	wg.Add(idRequired)

	uid, err := NewUnixID()
	if err != nil {
		t.Fatal(err)
		return
	}

	idObtained := make(map[string]int)
	var esperar sync.Mutex

	for i := 0; i < idRequired; i++ {
		go func() {
			defer wg.Done()

			id := uid.NewID()

			esperar.Lock()
			if cantId, exist := idObtained[id]; exist {
				idObtained[id] = cantId + 1
			} else {
				idObtained[id] = 1
			}
			esperar.Unlock()
		}()
	}
	wg.Wait()

	if idRequired != len(idObtained) {
		t.Fatalf("se esperaban: %d ids pero se obtuvieron: %d. Detalle: %v", idRequired, len(idObtained), idObtained)
	}
}

func BenchmarkGetNewID(b *testing.B) {
	uid, _ := NewUnixID()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uid.NewID()
	}
}

// Prueba adicional para verificar que no haya duplicados al generar muchos IDs
func TestNoDuplicateIDs(t *testing.T) {
	uid, err := NewUnixID()
	if err != nil {
		t.Fatal(err)
		return
	}

	numIDs := 1000
	ids := make(map[string]bool)

	for i := 0; i < numIDs; i++ {
		id := uid.NewID()
		if _, exists := ids[id]; exists {
			t.Fatalf("ID duplicado encontrado: %s", id)
		}
		ids[id] = true
	}
}

// Prueba para verificar que se generen IDs secuenciales cuando hay colisiones de timestamp
func TestSequentialIDs(t *testing.T) {
	uid, err := NewUnixID()
	if err != nil {
		t.Fatal(err)
		return
	}

	ids := make([]string, 10)
	for i := 0; i < 10; i++ {
		ids[i] = uid.NewID()
	}

	uniqueIDs := make(map[string]bool)
	for _, id := range ids {
		uniqueIDs[id] = true
	}

	if len(uniqueIDs) < len(ids) {
		t.Fatalf("Se esperaban %d IDs únicos, pero se obtuvieron %d", len(ids), len(uniqueIDs))
	}
}

func TestNewForReplica_Zero(t *testing.T) {
	uid, err := NewForReplica(0, 0)
	if uid != nil {
		t.Errorf("expected nil generator for replica 0")
	}
	if err == nil || err.Error() != ErrReplicaZero.Error() {
		t.Errorf("expected error %q, got %v", ErrReplicaZero.Error(), err)
	}
}

func TestNewForReplica_Suffix(t *testing.T) {
	uid, err := NewForReplica(42, 0)
	if err != nil {
		t.Fatal(err)
	}

	idStr := uid.NewID()
	timestamp, replica, err := uid.Parse(idStr)
	if err != nil {
		t.Fatalf("unexpected error parsing ID %s: %v", idStr, err)
	}
	if replica != 42 {
		t.Errorf("expected replica 42, got %d", replica)
	}
	if timestamp == 0 {
		t.Errorf("expected non-zero timestamp")
	}
}

func TestNewUnixID_Server(t *testing.T) {
	uid, err := NewUnixID()
	if err != nil {
		t.Fatal(err)
	}

	idStr := uid.NewID()
	timestamp, replica, err := uid.Parse(idStr)
	if err != nil {
		t.Fatalf("unexpected error parsing ID %s: %v", idStr, err)
	}
	if replica != 0 {
		t.Errorf("expected replica 0, got %d", replica)
	}
	if timestamp == 0 {
		t.Errorf("expected non-zero timestamp")
	}

	// Check for no '.' in server ID
	for _, c := range idStr {
		if c == '.' {
			t.Errorf("expected no '.' in server ID %s", idStr)
		}
	}
}

func TestMonotonic(t *testing.T) {
	uid, err := NewUnixID()
	if err != nil {
		t.Fatal(err)
	}

	var lastTimestamp int64
	for i := 0; i < 100000; i++ {
		idStr := uid.NewID()
		timestamp, _, err := uid.Parse(idStr)
		if err != nil {
			t.Fatalf("unexpected error parsing ID %s: %v", idStr, err)
		}
		if i > 0 && timestamp <= lastTimestamp {
			t.Fatalf("IDs not monotonic: previous %d, current %d", lastTimestamp, timestamp)
		}
		lastTimestamp = timestamp
	}
}

func TestConcurrency(t *testing.T) {
	uid, err := NewUnixID()
	if err != nil {
		t.Fatal(err)
	}

	numGoroutines := 8
	idsPerGoroutine := 10000

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	var mu sync.Mutex
	allIDs := make(map[string]bool)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			localIDs := make([]string, idsPerGoroutine)
			for j := 0; j < idsPerGoroutine; j++ {
				localIDs[j] = uid.NewID()
			}

			mu.Lock()
			for _, id := range localIDs {
				if allIDs[id] {
					t.Errorf("Duplicate ID found: %s", id)
				}
				allIDs[id] = true
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	if len(allIDs) != numGoroutines*idsPerGoroutine {
		t.Errorf("Expected %d IDs, but got %d", numGoroutines*idsPerGoroutine, len(allIDs))
	}
}

func TestLastHonoured(t *testing.T) {
	far := time.Now().UnixNano() + 3600e9
	uid, err := NewForReplica(7, far)
	if err != nil {
		t.Fatal(err)
	}

	idStr := uid.NewID()
	timestamp, replica, err := uid.Parse(idStr)
	if err != nil {
		t.Fatalf("unexpected error parsing ID %s: %v", idStr, err)
	}

	if replica != 7 {
		t.Errorf("expected replica 7, got %d", replica)
	}

	expectedTimestamp := far + 1
	if timestamp != expectedTimestamp {
		t.Errorf("expected timestamp %d, got %d", expectedTimestamp, timestamp)
	}
}
