package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"index/suffixarray"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sort"
)

func main() {
	searcher := Searcher{}
	err := searcher.Load("completeworks.txt")
	if err != nil {
		log.Fatal(err)
	}

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/search", handleSearch(searcher))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	fmt.Printf("Listening on port %s...", port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

type Searcher struct {
	CompleteWorks string
	SuffixArray   *suffixarray.Index
}

func handleSearch(searcher Searcher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query, ok := r.URL.Query()["q"]
		if !ok || len(query[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing search query in URL params"))
			return
		}
		results := searcher.Search(query[0])
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		err := enc.Encode(results)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("encoding failure"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(buf.Bytes())
	}
}

func (s *Searcher) Load(filename string) error {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	s.CompleteWorks = string(dat)
	r := strings.NewReader(strings.ToLower(s.CompleteWorks)) // converting the textfile to lowercase to get the case insensitive results

	b, err := ioutil.ReadAll(r)
	s.SuffixArray = suffixarray.New(b)
	return nil
}

func (s *Searcher) Search(query string) []string {
	queryCase := strings.ToLower(query)
	queryLen := len(query)
	idxs := s.SuffixArray.Lookup([]byte(queryCase), -1)
	sort.Sort(sort.IntSlice(idxs))
	results := []string{}
	docLength := len(s.CompleteWorks)
	idxsLength := len(idxs)
	for i, idx := range idxs {
		minRange := idx-250
		maxRange := idx+250
		if i > 0 {	// checking if it is not the 1st occurence
			if minRange <= idxs[i-1] {	// checking if the previous index is in the range of current minRange
				minRange = (idxs[i-1]+idx+queryLen)/2	// getting the minimum range if previous range is present for showing non-repetitive content
			}
		}
		if i+1 < idxsLength {	// checking if it is the last occurence
			if maxRange >= idxs[i+1] {	// checking if the next index is in the range of current maxRange
				maxRange = (idxs[i+1]+idx+queryLen)/2	// getting the maximum range if next range is present for showing non-repetitive content
			}
		}
		if minRange < 0 {	// checking if the staring range is 0
			minRange = 0
		}
		if maxRange > docLength {	// checking if the max range exceeding the doclength
			maxRange = docLength
		}
		results = append(results, s.CompleteWorks[minRange:maxRange])
	}
	return results
}
