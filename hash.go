package parser

//"fmt"

//"errors"
//"fmt"

type Hashable interface {
	Comparable
	HashCode() uint32
}

type Ordered interface {
	CompareOrder(v interface{}) int
}

type Comparable interface {
	Equals(v interface{}) bool
}

type Hashset interface {
	Size() int
	Has(x Hashable) (Hashable, bool)
	Add(x ...Hashable) int
	Replace(x ...Hashable) int
	AddReplace(x ...Hashable)
	Remove(x ...Hashable) int
	OpenCursor() Cursor
}

type Hashmap interface {
	ContainsKey(k Hashable) bool
	Get(k Hashable) (Comparable, bool)
	Put(k Hashable, v Comparable) bool
	Del(k Hashable, v Comparable) bool
	DelKey(k Hashable) int
	OpenCursor() Cursor
}

type Cursor interface {
	Next() interface{}
	HasMore() bool
	Close() error
}

///

type genericHashSet struct {
	x    map[uint32][]Hashable
	lock chan bool
	seq  chan int
	size int
}

func NewHashSet() Hashset {
	hs := &genericHashSet{
		x:    make(map[uint32][]Hashable),
		lock: make(chan bool, 1),
		seq:  make(chan int, 1),
	}
	hs.seq <- 0
	hs.lock <- true
	return hs
}

func (hs *genericHashSet) Size() int {
	<-hs.lock
	defer func() { hs.lock <- true }()
	return hs.size
}

func (hs *genericHashSet) Has(x Hashable) (Hashable, bool) {
	<-hs.lock
	defer func() { hs.lock <- true }()
	k := x.HashCode()
	if m, has := hs.x[k]; has {
		for _, v := range m {
			if x.Equals(v) {
				return v, true
			}
		}
	}
	return nil, false
}

func (hs *genericHashSet) Add(x ...Hashable) int {
	<-hs.lock
	nseq := <-hs.seq
	c := 0
	defer func() {
		if c > 0 {
			hs.seq <- nseq + 1
		} else {
			hs.seq <- nseq
		}
		hs.lock <- true
	}()
	for _, nv := range x {
		k := nv.HashCode()
		if m, has := hs.x[k]; has {
			for _, v := range m {
				if nv.Equals(v) {
					break
				}
			}
			hs.x[k] = append(hs.x[k], nv)
			hs.size++
			c++
		} else {
			hs.x[k] = []Hashable{nv}
			hs.size++
			c++
		}
	}
	return c
}

func (hs *genericHashSet) Replace(x ...Hashable) int {
	<-hs.lock
	nseq := <-hs.seq
	c := 0
	defer func() {
		if c > 0 {
			hs.seq <- nseq + 1
		} else {
			hs.seq <- nseq
		}
		hs.lock <- true
	}()
	for _, nv := range x {
		k := nv.HashCode()
		if m, has := hs.x[k]; has {
			for i, v := range m {
				if nv.Equals(v) {
					hs.x[k][i] = nv
					c++
				}
			}
		}
	}
	return c
}

func (hs *genericHashSet) AddReplace(x ...Hashable) {
	<-hs.lock
	nseq := <-hs.seq
	defer func() {
		hs.seq <- nseq + 1
		hs.lock <- true
	}()
	for _, nv := range x {
		k := nv.HashCode()
		if m, has := hs.x[k]; has {
			set := false
			for i, v := range m {
				if nv.Equals(v) {
					hs.x[k][i] = nv
					set = true
					break
				}
			}
			if !set {
				hs.size++
				hs.x[k] = append(hs.x[k], nv)
			}
		} else {
			hs.size++
			hs.x[k] = []Hashable{nv}
		}
	}
}

func (hs *genericHashSet) Remove(x ...Hashable) int {
	<-hs.lock
	nseq := <-hs.seq
	c := 0
	defer func() {
		if c > 0 {
			hs.seq <- nseq + 1
		} else {
			hs.seq <- nseq
		}
		hs.lock <- true
	}()
	for _, nv := range x {
		k := nv.HashCode()
		if m, has := hs.x[k]; has {
			for i, v := range m {
				if nv.Equals(v) {
					hs.x[k] = append(hs.x[k][0:i], hs.x[k][i+1:]...)
					hs.size--
					c++
					break
				}
			}
		}
	}
	return c
}

type genericHashSetCursor struct {
	set        *genericHashSet
	out        chan Hashable
	readReady  chan bool
	writeReady chan bool
	open       bool
	seq        int
}

func (hs *genericHashSet) OpenCursor() Cursor {
	cursor := &genericHashSetCursor{
		set:        hs,
		out:        make(chan Hashable, 1),
		readReady:  make(chan bool, 1),
		writeReady: make(chan bool, 1),
		open:       true,
	}
	<-hs.lock
	defer func() { hs.lock <- true }()
	cursor.seq = <-hs.seq
	hs.seq <- cursor.seq
	go func() {
		defer func() {
			cursor.open = false
			cursor.readReady <- true
		}()
		for _, m := range hs.x {
			for _, v := range m {
				cursor.out <- v
				cursor.readReady <- true
				<-cursor.writeReady
			}
		}
	}()
	return cursor
}

func (hsc *genericHashSetCursor) Next() interface{} {
	var value Hashable
	<-hsc.readReady
	if !hsc.open {
		hsc.readReady <- true
		panic("no more elements in iterator")
	}
	// Check for concurrent modification
	<-hsc.set.lock
	defer func() { hsc.set.lock <- true }()
	seq := <-hsc.set.seq
	hsc.set.seq <- seq
	if seq != hsc.seq {
		panic("concurrent modification")
	}
	select {
	case value = <-hsc.out:
	default:
		panic("no value provided after readReady signalled")
	}
	hsc.writeReady <- true
	return value
}

func (hsc *genericHashSetCursor) HasMore() bool {
	<-hsc.readReady
	defer func() { hsc.readReady <- true }()
	return hsc.open
}

func (hsc *genericHashSetCursor) Close() error {
	panic("unimplemented")
}
