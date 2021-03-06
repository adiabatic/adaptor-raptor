package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Warning struct {
	Haystack string
	At       int
	Length   int
	Message  string
}

type Notices struct {
	error
	Warnings []Warning
}

func (e *Notices) Error() string {
	var buffer bytes.Buffer
	for _, w := range e.Warnings {
		ctx := w.Haystack[w.At-10 : w.Length+10]
		msgPrefix := "ambiguous choice: «"
		msgSuffix := "»\n"
		buffer.WriteString(msgPrefix + ctx + msgSuffix)
		buffer.WriteString(strings.Repeat(" ", utf8.RuneCountInString(msgPrefix)+10) + "^\n\n")
	}

	return buffer.String()
}

func (e *Notices) AddWarning(w Warning) {
	if e.Warnings == nil {
		e.Warnings = make([]Warning, 1)
	}
	e.Warnings = append(e.Warnings, w)
}

type Entry struct {
	// The word written in Latin script.
	From string `yaml:"f"`

	// The word in Quikscript, assuming there’s only one spelling.
	// For strings of Latin text with multiple possible pronunciations (“use”, “arithmetic”, etc.),
	// this will be "", and you’ll need to look in .Possibilities.
	To string `yaml:"t"`

	// The Junior Quikscript expansion, if there is one. There’s about five.
	Junior string
	// The Senior Quikscript abbreviation, if there is one.
	Senior string

	// Some strings like "use" in Latin have multiple pronunciations depending on what part
	// of speech it’s supposed to be. For these words, Possibilities lists all the different
	// things it could be.
	//Possibilities []string

	CaseSensitive bool `yaml:"case-sensitive"`
	Warning       string

	// Higher-priority than entries that don’t have this set to true.
	HighPriority bool
}

type trieNode struct {
	// The value of the node’s key/value pair. Empty if this node isn’t a complete key.
	value Entry

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

func (t *trieNode) add(key string, val Entry, priority int, r *Replacer) {
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

func (r *Replacer) lookup(s string, ignoreRoot bool) (val Entry, keylen int, found bool) {
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
	root       trieNode
	tableSize  int
	mapping    [256]byte
	ignoreHTML bool
}

func NewReplacer(ignoreHTML bool, oldnew []Entry) *Replacer {
	r := new(Replacer)
	r.ignoreHTML = ignoreHTML

	// go through all the keys…
	//  and for every different byte in the keys…
	//      put a 1 in that byte’s index in the mapping.

	for i := 0; i < len(oldnew); i++ {
		key := oldnew[i].From
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
	for i := 0; i < len(oldnew); i++ {
		//                                    add things highest-priority first
		r.root.add(oldnew[i].From, oldnew[i], len(oldnew)-i, r)
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

func (r *Replacer) Replace(s string) (string, error) {
	buf := make(appendSliceWriter, 0, len(s))
	_, err := r.WriteString(&buf, s)
	return string(buf), err
}

// Returns true if a string starting at index i and length l is surrounded by non-letters.
// The beginning and end of strings are considered non-letters.
func isIsolatedWord(haystack string, i, l int) bool {
	alpha := rune(' ')
	if i > 0 {
		alpha, _ = utf8.DecodeRuneInString(haystack[i-1:])
	}
	omega, _ := utf8.DecodeRuneInString(haystack[i+l:])

	return !unicode.IsLetter(alpha) && !unicode.IsLetter(omega)
}

func (r *Replacer) WriteString(w io.Writer, s string) (n int, err error) {
	sw := getStringWriter(w)
	err = &Notices{}
	var last int // where the next copy-to should start from in s
	var wn int   // freshly-written (to w) n
	var prevMatchEmpty bool

	for i := 0; i <= len(s); {
		if r.ignoreHTML && i < len(s) {

			if s[i] == '<' {
				closingAngleBracketIndex := strings.IndexRune(s[i:], '>')
				if closingAngleBracketIndex != -1 {
					delta := closingAngleBracketIndex
					i += delta
					continue
				}
			}
		}

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

			wn, err = sw.WriteString(val.To)
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
