package nlock

import (
  "reflect"
  "testing"
  "time"
)

func TestLockUnlockSimple(t *testing.T) {
  ns := "/a/b/c"
  nlock := NewNamespaceLock()
  nlock.Lock(ns)
  if !nlock.lockHeld(ns, Write) {
    t.Errorf("Write lock should be held for: %v", ns)
  }
  nlock.Unlock(ns)
  if nlock.lockHeld(ns, Write) {
    t.Errorf("Write lock should not be held for: %v", ns)
  }
}

func TestLockUnlockReadParents(t *testing.T) {
  nlock := NewNamespaceLock()
  nlock.Lock("/a/b/c")
  nlock.RLock("/a/b")   // This should not block
  nlock.Unlock("/a/b/c")
  if !nlock.lockHeld("/a/b", Read) {
    t.Errorf("Read lock should be held for: %v", "/a/b")
  }
  nlock.RUnlock("/a/b")
  if nlock.lockHeld("/a/b", Read) {
    t.Errorf("Read lock should not be held for: %v", "/a/b")
  }
}

func TestLockUnlockBlockingWrite(t *testing.T) {
  nlock := NewNamespaceLock()
  ch := make(chan string)
  events := []string{}

  // Hold the lock for 10ms
  nlock.Lock("/a/b/c")

  go func() {
    nlock.Lock("/a/b/c")
    ch <- "acquire"
    nlock.Unlock("/a/b/c")
  }()

  go func() {
    select {
    case <-time.After(10 * time.Millisecond):
      ch <- "unlock"
      nlock.Unlock("/a/b/c")
    }
  }()

  events = append(events, <-ch)
  events = append(events, <-ch)

  expected := []string{"unlock", "acquire"}
  if !reflect.DeepEqual(events, expected) {
    t.Errorf("Invalid event ordering, got: %v, expected: %v", events, expected)
  }
}

func TestLockUnlockConcurrent(t *testing.T) {
  m := map[string]uint8{}
  writeCh := make(chan struct{})
  readCh  := make(chan struct{})
  ns := "/a/b/c"
  nlock := NewNamespaceLock()

  go func() {
    for i := 0; i < 100; i++ {
      nlock.Lock(ns)
      m[ns] = m[ns] + 1
      nlock.Unlock(ns)
    }
    close(writeCh)
  }()

  go func() {
    for i := 0; i < 100; i++ {
      nlock.RLock(ns)
      v := m[ns]
      nlock.RUnlock(ns)
      if v < 0 || v > 100 {
        t.Errorf("Invalid value read: %v", v)
      }
    }
    close(readCh)
  }()

  <-writeCh
  <-readCh

  if m[ns] != 100 {
    t.Errorf("Expected: %v, got: %v", 100, m[ns])
  }
}
