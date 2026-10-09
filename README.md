# UnixID
<img src="docs/img/badges.svg">

A Go library for generating unique, time-based IDs using Unix timestamps at nanosecond precision.

## Overview

UnixID provides functionality for generating and managing unique identifiers with the following features:

- High-performance ID generation based on Unix nanosecond timestamps
- Thread-safe concurrent ID generation
- Built-in collision avoidance through sequential numbering
- Date conversion utilities for timestamp-to-date formatting
- Versatile ID assignment for strings, struct fields and byte slices
- Explicit assignment for server and offline devices (replicas)

## Installation

```bash
go get webtyp.com/unixid
```

## Quick Start

### What generator should I use?

| I want... | Use... |
| --------- | ------ |
| Server ids | `NewUnixID()` |
| Offline device ids | `NewForReplica(replica, last)` |
| Read an id back | `Parse` |

### Server-side Usage

```go
package main

import (
	"fmt"
	"webtyp.com/unixid"
)

func main() {
	// Create a new UnixID handler (server-side)
	idHandler, err := unixid.NewUnixID()
	if err != nil {
		panic(err)
	}

	// Generate a new unique ID
	id := idHandler.NewID()

	fmt.Printf("Generated ID: %s\n", id)
	// Output: Generated ID: 1624397134562544800
}
```

### Offline Device (Replica) Usage

For offline devices, you provide a replica number (assigned by the server) and the last timestamp generated. The `last` parameter is necessary because a device restarted after its clock moved backwards must not repeat an id.

```go
package main

import (
	"fmt"
	"webtyp.com/unixid"
)

func main() {
	// Replicas must be > 0. A replica number of 0 returns an error.
	var myReplica unixid.Replica = 42

	// last is the timestamp part of the newest id this replica already minted.
	var lastIdTimestamp int64 = 0
	
	idHandler, err := unixid.NewForReplica(myReplica, lastIdTimestamp)
	if err != nil {
		panic(err)
	}
	
	id := idHandler.NewID()
	
	fmt.Printf("Generated ID: %s\n", id)
	// Output: Generated ID: 1624397134562544800.42
}
```

## ID Format

The generated IDs follow this format:

- Server-side: `[unix_timestamp_in_nanoseconds]` (e.g., `1624397134562544800`)
- Replica: `[unix_timestamp_in_nanoseconds].[replica]` (e.g., `1624397134562544800.42`)

IDs of one generator sort lexicographically while timestamps have 19 digits (until year 2286).

## API Reference

### Core Functions

- `NewUnixID()`: Creates a new UnixID handler for ID generation on servers. Server ids are generated without a suffix.
- `NewForReplica(replica Replica, last int64)`: Creates a UnixID handler for an offline device. `replica` must be > 0.
- `NewID()`: Generates a new unique ID and returns it as a string.

- `SetNewID(target *string)`: Generates a new unique ID and assigns it to target
  - Example usages:
    ```go
    // Set ID to a string variable
    var id string
    idHandler.SetNewID(&id)

    // Set ID to a struct field
    type User struct { ID string }
    user := User{}
    idHandler.SetNewID(&user.ID)
    ```

- `Validate(id string) error`: Validates the format of an ID string without parsing it
  - Returns error if format is invalid

- `Parse(id string) (timestamp int64, replica Replica, error)`: Parses an ID string and extracts its components
  - Validates format first, then extracts timestamp and optional replica
  - Returns timestamp as int64, replica as `Replica` (0 if minted by server)

## Validate and Parse IDs

The library provides two methods for working with existing IDs:

### Validation Only

Use `Validate()` when you only need to check if an ID format is valid:

```go
package main

import (
	"fmt"
	"webtyp.com/unixid"
)

func main() {
	idHandler, _ := unixid.NewUnixID()
	
	id := "1624397134562544800"
	err := idHandler.Validate(id)
	if err != nil {
		fmt.Println("Invalid ID format")
		return
	}
	
	fmt.Println("Valid ID format")
}
```

### Parsing ID Components

Use `Parse()` when you need to extract the timestamp and replica:

```go
package main

import (
	"fmt"
	"webtyp.com/unixid"
)

func main() {
	idHandler, _ := unixid.NewUnixID()
	
	// Parse server-side ID
	timestamp, replica, err := idHandler.Parse("1624397134562544800")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Timestamp: %d, Replica: %v\n", timestamp, replica)
	// Output: Timestamp: 1624397134562544800, Replica: 0
	
	// Parse offline device ID
	timestamp, replica, err = idHandler.Parse("1624397134562544800.42")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Timestamp: %d, Replica: %v\n", timestamp, replica)
	// Output: Timestamp: 1624397134562544800, Replica: 42
}
```

## [Contributing](https://github.com/webtyp/cdvelop/blob/main/CONTRIBUTING.md)
---
## [License](LICENSE)