# gopy

Call Python directly from Go without running a separate Python service. `gopy` embeds Python source into your Go binary, starts managed Python worker processes, and supports an easy, low effort interface to call to python.

Go is a great primary server language, but it is often useful to delegate to Python for tasks like ML/data processing where the ecosystem is stronger. `gopy` lets you keep a single Go application while still calling Python functions efficiently.

## Install

Add the Go package:

```bash
go get github.com/jptrs93/gopy/gopy
```

Install the Python adapter in your Python environment:

```bash
pip install gopyadapter
```

## Usage

In a typical project structure, you can keep your Python and Go source in sibling directories. Then use `go:generate` to copy your Python source into an embeddable directory under your Go app during build.

Example structure:

```text
.
├── pysrc/
│   ├── pypkg/
│   │   ├── helper.py
│   │   └── helper2.py
│   ├── pyproject.toml
│   └── main.py
└── goapp/
    ├── otherpkg/
    ├── python/
    │   ├── pysrc/         # generated copy; untracked
    │   └── python.go
    └── main.go
```
If you prefer a simpler structure, you can also work directly with your Python source in the embeddable directory within the Go source.

### `pysrc/main.py`

```python
import numpy as np
from gopyadapter.core import execute
from pypkg.helper import weighted_total
from pypkg.helper2 import normalize_columns


def summarize_customer(i):
    profile = i["profile"]
    transactions = np.asarray(i["transactions"], dtype=np.float64)
    weights = np.asarray(i["weights"], dtype=np.float64)

    total = weighted_total(transactions, weights)
    avg = float(np.mean(transactions))

    return {
        "id": profile["id"],
        "name": profile["name"],
        "tier": profile["metadata"]["tier"],
        "weightedTotal": total,
        "averageTransaction": avg,
    }


def normalize_matrix(i):
    matrix = np.asarray(i["matrix"], dtype=np.float64)
    normalized = normalize_columns(matrix)

    return {
        "shape": [int(matrix.shape[0]), int(matrix.shape[1])],
        "normalized": normalized,
    }


if __name__ == "__main__":
    execute(**globals())
```

### `goapp/python/python.go`

```go
package python

import (
	"embed"
)

// Copy editable Python files into an embeddable folder.
//go:generate sh -c "rm -rf ./pysrc && cp -R ../../pysrc ./pysrc"

//go:embed pysrc/*
var PythonSrc embed.FS
```

### `goapp/main.go`

```go
package main

import (
	"fmt"

	"github.com/jptrs93/gopy/gopy"
	"yourmodule/goapp/python"
)

type CustomerMetadata struct {
	Tier string `msgpack:"tier,omitempty"`
}

type CustomerProfile struct {
	ID       int              `msgpack:"id,omitempty"`
	Name     string           `msgpack:"name,omitempty"`
	Metadata CustomerMetadata `msgpack:"metadata,omitempty"`
}

type SummarizeCustomerInput struct {
	Profile      CustomerProfile   `msgpack:"profile,omitempty"`
	Transactions gopy.Float64_Array `msgpack:"transactions,omitempty"`
	Weights      gopy.Float64_Array `msgpack:"weights,omitempty"`
}

type SummarizeCustomerResult struct {
	ID                 int     `msgpack:"id,omitempty"`
	Name               string  `msgpack:"name,omitempty"`
	Tier               string  `msgpack:"tier,omitempty"`
	WeightedTotal      float64 `msgpack:"weightedTotal,omitempty"`
	AverageTransaction float64 `msgpack:"averageTransaction,omitempty"`
}

type NormalizeMatrixInput struct {
	Matrix gopy.Float64_2DArray `msgpack:"matrix,omitempty"`
}

type NormalizeMatrixResult struct {
	Shape      []int               `msgpack:"shape,omitempty"`
	Normalized gopy.Float64_2DArray `msgpack:"normalized,omitempty"`
}

func main() {
	// Run once before build/run.
	// (from goapp/) go generate ./...

	gopy.InitDefaultPool(
		python.PythonSrc,
		"/path-to-python-env/bin/python",
		"main.py",
		2,
	)

	customerRes, err := gopy.CallDefault[SummarizeCustomerResult](
		"summarize_customer",
		SummarizeCustomerInput{
			Profile: CustomerProfile{
				ID:   42,
				Name: "Alex",
				Metadata: CustomerMetadata{
					Tier: "gold",
				},
			},
			Transactions: gopy.Float64_Array{19.99, 45.10, 88.00},
			Weights:      gopy.Float64_Array{0.2, 0.3, 0.5},
		},
	)
	if err != nil {
		panic(err)
	}

	matrixRes := gopy.MustCallDefault[NormalizeMatrixResult](
		"normalize_matrix",
		NormalizeMatrixInput{
			Matrix: gopy.Float64_2DArray{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}},
		},
	)

	fmt.Printf("customer result: %+v\n", customerRes)
	fmt.Printf("matrix shape: %v\n", matrixRes.Shape)
}
```

Run:

```bash
go generate ./...
go run .
```

> [!NOTE]
> The Python runtime/environment must be set up with the correct dependencies installed for your Python code and be available on the same machine where the Go application runs. The path to the Python executable must be passed when initializing the `gopy.Pool` in Go.

### Pool lifecycle

1. Using the default pool

```go
gopy.InitDefaultPool(python.PythonSrc, "/path-to-python-env/bin/python", "main.py", 2)

res, err := gopy.CallDefault[SummarizeCustomerResult]("summarize_customer", input)
```

Use this when your app has one shared Python runtime configuration and the pool lifecycle matches the Go application lifecycle.

2. Creating a pool instance, passing it around, and closing it

```go
package service

import (
	"context"
	"github.com/jptrs93/gopy/gopy"
	"yourmodule/goapp/python"
)

func Example() {
	pool := gopy.NewPool(context.Background(), python.PythonSrc, "/path-to-python-env/bin/python", "main.py", 2)
	defer pool.Close()

	res, err := gopy.CallPool[SummarizeCustomerResult](pool, "summarize_customer", input)

}
```
## Design

The `gopy` package in your Go application manages a pool of child Python processes (1 in the default case). The processes are tied to the lifecycle of the parent app and end if the main Go application is terminated.

<p align="center">
  <img src="./diagrams/gopyprocessdesign.drawio.svg" width="75%">
</p>
<p align="center"><em>Figure: gopy Process Design</em></p>

Communication between Go and Python uses a simple protocol over 2 additional custom pipes. The Python process stdout and stderr streams are consumed and written to logs in Go, but are not used as part of the protocol.

<p align="center">
  <img src="./diagrams/gopypipes.drawio.svg" width="60%">
</p>
<p align="center"><em>Figure: gopy Pipes</em></p>

Protocol:

1. Go writes:
   1. Length-prefixed function name (4-byte big-endian length + bytes)
   2. Length-prefixed payload (4-byte big-endian length + MessagePack bytes)
2. Python reads and decodes function name and payload, then calls that function with the payload.
3. Python encodes and writes the result as a length-prefixed payload (4-byte big-endian length + MessagePack bytes).
4. Go reads and decodes the result.

Calls are processed sequentially per Python worker process. You can safely call a Python function from another goroutine while an existing function is running; however, it will be blocked until the first call finishes.
