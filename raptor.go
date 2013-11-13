package main

import (
	//	"bytes"
	"fmt"
	"io/ioutil"
	"launchpad.net/goyaml"
	"strings"
)

func main() {

	overrides, err := ioutil.ReadFile("00.yaml")
	if err != nil {
		panic("couldn’t read overrides file")
	}

	normal_words, err := ioutil.ReadFile("50.yaml")
	if err != nil {
		panic("couldn’t read the large chunk of normal words")
	}

	corpus := strings.Join([]string{string(overrides), string(normal_words)}, "")

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
	}

	replacer := New(replacements)
	s, err := replacer.Replace(string(document))
	fmt.Println(s)
	if err != nil {

	}

}
