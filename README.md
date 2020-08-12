Namespace Lock
==============

Namespace locking implementation in Go. A namespace is a hierarchical resource
structure that can be thought of as a tree similar to directory structure in
Unix system.

## Usage

```go
package main

import (
  "github.com/aprimadi/namespace-lock"
)

func main() {
  lock := nlock.NewNamespaceLock()

  go func() {
    // Acquire a read lock on "/a", "/a/b"
    lock.RLock("/a/b")

    // ... do things

    // Release read locks "/a" and "/a/b"
    lock.RUnlock("/a/b")
  }()

  go func() {
    // Acquire a read lock on "/a", "/a/b" and a write lock on "/a/b/c"
    lock.Lock("/a/b/c")

    // ... do things

    // Release read locks "/a", "/a/b" and write lock "/a/b/c"
    lock.Unlock("/a/b/c")
  }()
}
```
