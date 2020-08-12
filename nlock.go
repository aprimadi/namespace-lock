package nlock

import (
  "fmt"
  "sort"
  "strings"
  "sync"
)

type NamespaceLock struct {
  mu sync.Mutex         // Protect the map below
  locks map[node]*lock
}

type Mode uint8

const (
  Read Mode = iota
  Write
)

func NewNamespaceLock() *NamespaceLock {
  n := new(NamespaceLock)
  n.locks = make(map[node]*lock)
  return n
}

func (n *NamespaceLock) Lock(namespace string) {
  n.lock(namespace, Write)
}

func (n *NamespaceLock) Unlock(namespace string) {
  n.unlock(namespace, Write)
}

func (n *NamespaceLock) RLock(namespace string) {
  n.lock(namespace, Read)
}

func (n *NamespaceLock) RUnlock(namespace string) {
  n.unlock(namespace, Read)
}

func (n *NamespaceLock) lock(namespace string, mode Mode) {
  nodes := split(namespace)

  for i, node := range nodes {
    m := Read
    if i == len(nodes) - 1 {
      m = mode
    }

    n.acquire(node, m)
  }
}

func (n *NamespaceLock) unlock(namespace string, mode Mode) {
  nodes := split(namespace)
  sort.Sort(sort.Reverse(nodes))

  for i, node := range nodes {
    m := Read
    if i == 0 {
      m = mode
    }

    n.release(node, m)
  }
}

func (n *NamespaceLock) acquire(node node, mode Mode) {
  // Attempt to acquire and block on the channel until it's ready to acquire
  for {
    var l *lock
    var ok bool

    n.mu.Lock()

    if l, ok = n.locks[node]; !ok {
      // Not found, create a new lock
      l = new(lock)
      l.notifyCh = make(chan struct{})
      n.locks[node] = l
    }

    if mode == Read && l.writers == 0 {
      l.readers++
      n.mu.Unlock()
      return
    } else if mode == Write && l.readers == 0 && l.writers == 0 {
      l.writers++
      n.mu.Unlock()
      return
    }

    n.mu.Unlock()

    // Wait until someone release the lock to check again
    <-l.notifyCh
  }
}

func (n *NamespaceLock) release(node node, mode Mode) {
  n.mu.Lock()
  defer n.mu.Unlock()

  l := n.locks[node]

  if mode == Read {
    if l.readers == 0 {
      panic(fmt.Sprintf("RUnlock of unlocked node: %v", node))
    }
    l.readers--
  } else {
    if l.writers == 0 {
      panic(fmt.Sprintf("Unlock of unlocked node: %v", node))
    }
    l.writers--
  }

  // Remove lock from map
  if l.readers == 0 && l.writers == 0 {
    delete(n.locks, node)
  }

  ch := l.notifyCh
  l.notifyCh = make(chan struct{})
  close(ch)
}

// Is the lock held? Only used for testing
func (n *NamespaceLock) lockHeld(namespace string, mode Mode) bool {
  nodes := split(namespace)

  result := true

  n.mu.Lock()

loop:
  for i, node := range nodes {
    m := Read
    if i == len(nodes) - 1 {
      m = mode
    }

    l := n.locks[node]
    switch {
    case l == nil:
      fallthrough
    case m == Read && l.readers == 0:
      fallthrough
    case m == Write && l.writers == 0:
      result = false
      break loop
    }
  }

  n.mu.Unlock()

  return result
}

type lock struct {
  readers uint32
  writers uint32
  notifyCh chan struct{}
}

type node string

type nodes []node
func (n nodes) Len() int            { return len(n) }
func (n nodes) Swap(i, j int)       { n[i], n[j] = n[j], n[i] }
func (n nodes) Less(i, j int) bool  { return n[i] < n[j] }

func split(namespace string) nodes {
  nodes := []node{}
  tokens := strings.Split(namespace[1:], "/")
  for i := range tokens {
    nodes = append(nodes, (node)("/" + strings.Join(tokens[:i+1], "/")))
  }
  return nodes
}
