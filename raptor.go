package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"launchpad.net/goyaml"
	"strings"
	// "log"
	"os"
)

const DEBUG = false

func entriesFromFile(filename string, highPriority bool) []Entry {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		panic("couldn’t read dictionary file: " + filename)
	}

	// This will work OK as long as there isn't a "---" anywhere in the file
	// other than in document separators.
	sep := []byte("---")
	contents_split := bytes.Split(contents, sep)

	entries := make([]Entry, 0, 1024)

	for _, textEntry := range contents_split {
		e := Entry{}
		err := goyaml.Unmarshal(textEntry, &e)
		if err != nil {
			panic(err)
		}

		// filter out purely informative “entries” and entries with one-to-many mappings
		if e.From == "" || e.To == "" {
			continue
		}

		e.HighPriority = highPriority

		if !e.CaseSensitive {
			f := e
			f.From = strings.Title(f.From)
			entries = append(entries, f)
		}

		entries = append(entries, e)
	}
	return entries
}

func assemble_corpus(files ...string) string {
	ss := make([]string, 0, 3)
	for _, fn := range files {
		contents, err := ioutil.ReadFile(fn)
		if err != nil {
			panic("couldn’t read dictionary file: " + fn)
		}
		ss = append(ss, string(contents))
	}

	return strings.Join(ss, "")
}

func main() {
	document, err := ioutil.ReadFile("source.markdown")
	if err != nil {
		panic("couldn’t read source markdown text")
	}

	highs := entriesFromFile("04.yaml", true)
	mediums := entriesFromFile("50.yaml", false)
	replacements := append(highs, mediums...)

	By(zToAByFrom).Sort(replacements)

	if DEBUG {
		for _, e := range replacements {
			fmt.Fprintf(os.Stderr, "%#v\n", e)
		}
	}

	replacer := NewReplacer(true, replacements)
	s, err := replacer.Replace(string(document))
	fmt.Println(s)
	if err != nil {

	}

}
