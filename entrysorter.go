package main

import (
	"sort"
	"strings"
)

type By func(l, r *Entry) bool

type entrySorter struct {
	entries []Entry
	by      By
}

func (es *entrySorter) Len() int {
	return len(es.entries)
}

func (es *entrySorter) Swap(i, j int) {
	es.entries[i], es.entries[j] = es.entries[j], es.entries[i]
}

func (es *entrySorter) Less(i, j int) bool {
	return es.by(&es.entries[i], &es.entries[j])
}

func (by By) Sort(entries []Entry) {
	es := &entrySorter{
		entries: entries,
		by:      by,
	}
	sort.Sort(es)
}

func zToAByFrom(l, r *Entry) bool {
	return strings.ToLower(l.From) > strings.ToLower(r.From)
}
