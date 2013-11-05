package main

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "launchpad.net/goyaml"

//    "strings"
)

type Entry struct {
    From          string `yaml:"f"`
    To            string `yaml:"t"`
    CaseSensitive bool   `yaml:"case-sensitive"`
    Warning       string
}

func main() {
    corpus, err := ioutil.ReadFile("en-Latn2en-Qaas.yaml")
    if err != nil {
        panic("couldn’t read corpus")
    }

    document, err := ioutil.ReadFile("source.markdown")
    if err != nil {
        panic("couldn’t read source markdown text")
    }

    // This will work OK as long as there isn't a "---" anywhere in the file
    // other than in document separators.
    sep := []byte("---")
    corpus_split := bytes.Split(corpus, sep)

    replacements := make([]string, 0, 1024)

    for _, entry := range corpus_split {
//        fmt.Printf("%v", string(entry))

        e := Entry{}
        err := goyaml.Unmarshal(entry, &e)
        if err != nil {
            panic(err)
        }

        replacements = append(replacements, e.From, e.To)
//        fmt.Printf("%#v\n", e)
    }

    replacer := New(replacements...)
    s := replacer.Replace(string(document))
    fmt.Println(s)

}
