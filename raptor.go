package main

import (
	//	"bytes"
	"fmt"
	"io/ioutil"
	"launchpad.net/goyaml"
	"strings"
	// "log"
)

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
	corpus := assemble_corpus("00.yaml", "04.yaml", "50.yaml")

	document, err := ioutil.ReadFile("source.markdown")
	if err != nil {
		panic("couldn’t read source markdown text")
	}

	// This will work OK as long as there isn't a "---" anywhere in the file
	// other than in document separators.
	sep := "---"
	corpus_split := strings.Split(corpus, sep)

	replacements := make([]Entry, 0, 1024)

	for _, entry := range corpus_split {
		//        fmt.Printf("%v", string(entry))

		e := Entry{}
		err := goyaml.Unmarshal([]byte(entry), &e)
		if err != nil {
			panic(err)
		}

		// filter out purely informative “entries” and entries with one-to-many mappings
		if e.From == "" || e.To == "" {
			continue
		}

		replacements = append(replacements, e)

		if !e.CaseSensitive {
			f := e
			f.From = strings.Title(f.From)
			replacements = append(replacements, f)
		}
	}

	By(zToAByFrom).Sort(replacements)

	replacer := NewReplacer(true, replacements)
	s, err := replacer.Replace(string(document))
	fmt.Println(s)
	if err != nil {

	}

}
