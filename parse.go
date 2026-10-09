package unixid

import (
	. "webtyp.com/fmt"
)

// Parse parses an ID string and extracts its components.
// It first validates the ID format, then extracts the timestamp and optional replica.
//
// Parameters:
//   - id: The ID string to parse (e.g., "1624397134562544800" or "1624397134562544800.42")
//
// Returns:
//   - timestamp: The timestamp portion as int64
//   - replica: The replica portion as Replica (0 if not present)
//   - err: An error if the ID format is invalid or parsing fails
func (u *UnixID) Parse(id string) (timestamp int64, replica Replica, err error) {
	// Primero valida el formato
	if err := u.Validate(id); err != nil {
		return 0, 0, err
	}

	// Encuentra el índice del punto (si existe)
	point_index := len(id)
	for i, char := range id {
		if char == '.' {
			point_index = i
			break
		}
	}

	// Extrae la parte del timestamp
	timestamp_str := id[:point_index]

	// Convierte el timestamp a int64
	timestamp, er := Convert(timestamp_str).Int64()
	if er != nil {
		return 0, 0, Err("format", "invalid")
	}

	// Extrae el replica si existe
	if point_index < len(id)-1 {
		replicaStr := id[point_index+1:]
		replicaUint, er := Convert(replicaStr).Uint32()
		if er != nil || replicaUint == 0 {
			return 0, 0, Err("format", "invalid")
		}
		replica = Replica(replicaUint)
	}

	return timestamp, replica, nil
}
