package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

func SingleHash(in, out chan interface{}) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := range in {
		wg.Add(1)
		go func(wg *sync.WaitGroup, mu *sync.Mutex, i interface{}, out chan interface{}) {
			defer wg.Done()
			input := i.(int)
			str := strconv.Itoa(input)
			mu.Lock()
			md5 := DataSignerMd5(str)
			mu.Unlock()
			chCrc := make(chan string)
			chMd5 := make(chan string)
			go func(str string, ch chan<- string) {
				res := DataSignerCrc32(str)
				ch <- res
			}(str, chCrc)
			go func(str string, ch chan<- string) {
				res := DataSignerCrc32(str)
				ch <- res
			}(md5, chMd5)
			res := <-chCrc
			resMd5 := <-chMd5
			resSingle := res + "~" + resMd5
			out <- resSingle
		}(&wg, &mu, i, out)
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	var wgMultiHash sync.WaitGroup

	for i := range in {
		wgMultiHash.Add(1)
		go func(wgMultiHash *sync.WaitGroup, i interface{}, out chan interface{}) {
			defer wgMultiHash.Done()
			str := i.(string)

			sl := make([]string, 6)
			wg := sync.WaitGroup{}
			mu := sync.Mutex{}

			for j := 0; j < 6; j++ {
				wg.Add(1)
				go func(slice []string, index int, step string, wg *sync.WaitGroup, mu *sync.Mutex) {
					defer wg.Done()
					th := strconv.Itoa(index)
					res := DataSignerCrc32(th + step)
					mu.Lock()
					slice[index] = res
					mu.Unlock()
				}(sl, j, str, &wg, &mu)
			}
			wg.Wait()
			res := strings.Join(sl, "")
			out <- res
		}(&wgMultiHash, i, out)
	}
	wgMultiHash.Wait()
}

func CombineResults(in, out chan interface{}) {
	var slice []string

	for data := range in {
		slice = append(slice, data.(string))
	}
	sort.Strings(slice)
	res := strings.Join(slice, "_")
	out <- res
}

func worker(wg *sync.WaitGroup, jobs job, in, out chan interface{}) {
	defer wg.Done()
	jobs(in, out)

	close(out)
}

func ExecutePipeline(jobs ...job) {
	in := make(chan interface{})
	out := make(chan interface{})
	wg := sync.WaitGroup{}

	for _, data := range jobs {
		wg.Add(1)
		go worker(&wg, data, in, out)
		in = out
		out = make(chan interface{})
	}
	wg.Wait()
}
