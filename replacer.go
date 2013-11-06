package main

import (
	"io"
	"strings"
	"unicode"
	"fmt"
	"unicode/utf8"
)

type trieNode struct {
	// The value of the node’s key/value pair. Empty if this node isn’t a complete key.
	value string

	// Priority of the pair; keys aren’t necessarily matched longest- or shortest-first.
	// This number is positive if this node is a complete key, and 0 otherwise.
	priority int

	// if this is true, then only exact matches should match this particular
	// entry in the trie.
	caseSensitive bool

	// Prefixes are used for the case where there’s only one child.
	// (more than one child? Use .table.)
	prefix string
	next   *trieNode
	table  []*trieNode
}

func (t *trieNode) add(key, val string, priority int /*, caseSensitive bool*/, r *Replacer) {
	// fail if someone tries to stuff nothing in as a key
	if key == "" {
		if t.priority == 0 {
			t.value = val
			t.priority = priority
		}
		return
	}

	// What, the current node has a prefix and we need to add in something else here?
	// That means we need to break the prefix up and spread it over multiple nodes.
	if t.prefix != "" {
		var n int // length of the longest common prefix
		for ; n < len(t.prefix) && n < len(key); n++ {
			if t.prefix[n] != key[n] {
				break
			}
		}

		if n == len(t.prefix) { // if the new key is at least as long as the prefix…
			t.next.add(key[n:], val, priority, r)
		} else if n == 0 {
			// the first byte differs between the prefix and the new key, so we’re going to set up
			// a new table here.
			var prefixNode *trieNode
			if len(t.prefix) == 1 {
				prefixNode = t.next
			} else {
				prefixNode = &trieNode{
					prefix: t.prefix[1:],
					next:   t.next,
				}
			}

			keyNode := new(trieNode)
			t.table = make([]*trieNode, r.tableSize)
			t.table[r.mapping[t.prefix[0]]] = prefixNode
			t.table[r.mapping[key[0]]] = keyNode
			t.prefix = ""
			t.next = nil
			keyNode.add(key[1:], val, priority, r)
		} else {
			// insert a new node after the common section of the prefix
			next := &trieNode{
				prefix: t.prefix[n:],
				next:   t.next,
			}
			t.prefix = t.prefix[:n]
			t.next = next
			next.add(key[n:], val, priority, r)
		}
	} else if t.table != nil {
		// we’ve got an existing table; shove it in
		m := r.mapping[key[0]]
		if t.table[m] == nil {
			t.table[m] = new(trieNode)
		}
		t.table[m].add(key[1:], val, priority, r)
	} else {
		t.prefix = key
		t.next = new(trieNode)
		t.next.add("", val, priority, r)
	}
}

func (r *Replacer) lookup(s string, ignoreRoot bool) (val string, keylen int, found bool) {
	// go down the trie to the end, and grab the val/keylen with the highest priority.
	bestPriority := 0
	node := &r.root
	n := 0
	for node != nil {
		if node.priority > bestPriority && !(ignoreRoot && node == &r.root) {
			bestPriority = node.priority
			val = node.value
			keylen = n
			found = true
		}

		if s == "" {
			break
		}

		if node.table != nil {
			index := r.mapping[s[0]]
			if int(index) == r.tableSize {
				break
			}

			node = node.table[index]
			s = s[1:]
			n++
		} else if node.prefix != "" && strings.HasPrefix(s, node.prefix) {
			n += len(node.prefix)
			s = s[len(node.prefix):]
			node = node.next
		} else {
			break
		}
	}
	return
}

type Replacer struct {
	root      trieNode
	tableSize int
	mapping   [256]byte
}

func New(oldnew ...string) *Replacer {

	if len(oldnew)%2 == 1 {
		panic("replacer.New: odd argument count")
	}

	r := new(Replacer)

	// go through all the keys…
	//  and for every different byte in the keys…
	//      put a 1 in that byte’s index in the mapping.
	for i := 0; i < len(oldnew); i += 2 {
		key := oldnew[i]
		for j := 0; j < len(key); j++ {
			r.mapping[key[j]] = 1
		}
	}

	// the tableSize, unsurprisingly, is the number of different bytes that’re used
	// in all the keys.
	for _, b := range r.mapping {
		r.tableSize += int(b)
	}

	// In order to do less work at replace time, we’re going to set up r.mapping.
	// It performs “oh, this is the byte you’re looking at? You want this entry in
	// trieNode.table.” functionality.
	var index byte
	for i, b := range r.mapping {
		// if an entry in the mapping table has nothing in it, set it to
		// tableSize (one past the end of the dense index)
		if b == 0 {
			r.mapping[i] = byte(r.tableSize)
		} else {
			r.mapping[i] = index
			index++
		}
	}

	// Not all nodes will want a lookup table, but the root node sure as heck will.
	r.root.table = make([]*trieNode, r.tableSize)

	// finally, add ’em all in.
	for i := 0; i < len(oldnew); i += 2 {
		//                                 add things highest-priority first
		r.root.add(oldnew[i], oldnew[i+1], len(oldnew)-i, r)
	}
	return r
}

type appendSliceWriter []byte

// satisfy io.Writer
func (w *appendSliceWriter) Write(p []byte) (int, error) {
	*w = append(*w, p...)
	return len(p), nil
}

func (w *appendSliceWriter) WriteString(s string) (int, error) {
	*w = append(*w, s...)
	return len(s), nil
}

type stringWriterIface interface {
	WriteString(string) (int, error)
}

type stringWriter struct {
	w io.Writer
}

func (w stringWriter) WriteString(s string) (int, error) {
	return w.w.Write([]byte(s))
}

func getStringWriter(w io.Writer) stringWriterIface {
	sw, ok := w.(stringWriterIface)
	if !ok {
		sw = stringWriter{w}
	}
	return sw
}

func (r *Replacer) Replace(s string) string {
	buf := make(appendSliceWriter, 0, len(s))
	r.WriteString(&buf, s)
	return string(buf)
}

// Returns true if a string starting at index i and length l is surrounded by non-letters.
// The beginning and end of strings are considered non-letters.
func isIsolatedWord(haystack string, i, l int) bool {
    alpha := rune(' ')
    if i > 0 {
        alpha, _ = utf8.DecodeRuneInString(haystack[i-1:])
    }
    omega, _ := utf8.DecodeRuneInString(haystack[i+l:])
    
    isolatedBeginning := unicode.IsSpace(alpha) || unicode.IsPunct(alpha)
    isolatedEnd := unicode.IsSpace(omega) || unicode.IsPunct(omega)

    return isolatedBeginning && isolatedEnd
}

func (r *Replacer) WriteString(w io.Writer, s string) (n int, err error) {
	sw := getStringWriter(w)
	var last, wn int
	var prevMatchEmpty bool

	for i := 0; i <= len(s); {
		// ignore the empty match iff the previous loop found the empty match
		val, keylen, match := r.lookup(s[i:], prevMatchEmpty)
		prevMatchEmpty = match && keylen == 0
        
        //fmt.Println(val)
		// first, let's make sure this thing ends on a word boundary. If it doesn't, then
		// screw it.
		if !isIsolatedWord(s, i, keylen) {
            i++
            continue
		}

		if match {
			wn, err = sw.WriteString(s[last:i])
			n += wn
			if err != nil {
				return
			}

			wn, err = sw.WriteString(val)
			n += wn
			if err != nil {
				return
			}
			i += keylen
			last = i
			continue
		}
		i++
	}
	if last != len(s) {
		wn, err = sw.WriteString(s[last:])
		n += wn
	}
	return
}

func dummy() {
	r, _ := utf8.DecodeRuneInString("z")
	_ = unicode.IsSpace(r)
	fmt.Println(r)
}
