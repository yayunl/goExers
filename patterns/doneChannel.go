package patterns

import (
	"fmt"
	"os"
	"time"
)

func SearchData(done <-chan struct{}, s string, searchers <-chan func(string, int) []string) <-chan []string {
	resultStream := make(chan []string)
	go func(s string) {
		defer close(resultStream)
		defer fmt.Println("searchData closure exited.")
		i := 0
		for search := range searchers {
			select {
			case <-done:
				return
			case resultStream <- search(s, i):
			}
			i += 1
		}
	}(s)
	return resultStream
}

func Generator(done <-chan struct{}, searchFuns ...func(string, int) []string) <-chan func(string, int) []string {
	dataStream := make(chan func(string, int) []string)
	go func() {
		defer close(dataStream)
		defer fmt.Println("Generator closure exited.")
		for _, searchFun := range searchFuns {
			select {
			case <-done:
				return
			case dataStream <- searchFun:
			}
		}
	}()

	return dataStream
}

func GetSearchFuns() []func(string, int) []string {
	sf := func(s string, i int) []string {
		time.Sleep(time.Duration(i) * time.Second)
		v := fmt.Sprintf("Searcher %d executed. Slept for %d secs", i, i)
		res := []string{s, v}
		return res
	}

	searchFuns := []func(string, int) []string{}
	for i := 0; i < 5; i++ {
		searchFuns = append(searchFuns, sf)
	}
	return searchFuns
}

func DemoSearchData() {
	done := make(chan struct{})
	searchFuns := GetSearchFuns()
	results := SearchData(done, "searchIt!", Generator(done, searchFuns...))

	for result := range results {
		fmt.Println(result)
		if result != nil {
			close(done)
			break
		}
	}
	time.Sleep(2 * time.Second)
	os.Exit(1)

}
