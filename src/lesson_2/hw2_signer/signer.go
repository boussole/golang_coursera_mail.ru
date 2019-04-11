package main

import (
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

func ExecutePipeline(jobs ...job) {
	in := make(chan interface{})

	for _, jb := range jobs {
		out := make(chan interface{})

		go func(in, out chan interface{}, jb job) {
			defer close(out)
			jb(in, out)
		}(in, out, jb)

		in = out
	}

	<-in
}

// prevents reordering of input args
func waitSelfOrder(selfSeqNo uint32, currentSeqNo *uint32) {
	for selfSeqNo != atomic.LoadUint32(currentSeqNo) {
		runtime.Gosched()
	}
}

func SingleHash(in, out chan interface{}) {
	var inSeqNo uint32
	var outSeqNo uint32 = 1
	wg := &sync.WaitGroup{}

	for input := range in {
		var val int
		var ok bool
		if val, ok = input.(int); !ok {
			panic("SingleHash: invalid input, int awaited")
		}

		inSeqNo++
		strVal := strconv.Itoa(val)
		wg.Add(1)

		go func(strVal string, md5Val string, out chan interface{}, wg *sync.WaitGroup, inSeqNo uint32, outSeqNo *uint32) {
			defer wg.Done()

			parallelResult := make(chan string)
			go func(strVal string, out chan<- string) {
				out <- "~" + DataSignerCrc32(md5Val)
			}(md5Val, parallelResult)

			// use current goroutine
			output := DataSignerCrc32(strVal)
			waitSelfOrder(inSeqNo, outSeqNo)
			out <- output + <-parallelResult
			atomic.AddUint32(outSeqNo, 1)
		}(strVal, DataSignerMd5(strVal), out, wg, inSeqNo, &outSeqNo)
	}

	wg.Wait()
}

func handleBatch(strVal string, out chan<- interface{}, wg *sync.WaitGroup, inSeqNo uint32, outSeqNo *uint32) {
	defer wg.Done()
	subWg := &sync.WaitGroup{}
	results := make([]string, 6, 6)
	for th := 1; th < 6; th++ {
		subWg.Add(1)
		go func(strVal string, resultIndex int, results []string, wg *sync.WaitGroup) {
			defer wg.Done()
			results[resultIndex] = DataSignerCrc32(strconv.Itoa(resultIndex) + strVal)
		}(strVal, th, results, subWg)
	}

	// use current goroutine
	results[0] = DataSignerCrc32("0" + strVal)
	waitSelfOrder(inSeqNo, outSeqNo)
	subWg.Wait()
	out <- strings.Join(results, "")
	atomic.AddUint32(outSeqNo, 1)
}

func MultiHash(in, out chan interface{}) {
	var inSeqNo uint32
	var outSeqNo uint32 = 1
	wg := &sync.WaitGroup{}
	for input := range in {
		var strVal string
		var ok bool

		if strVal, ok = input.(string); !ok {
			panic("MultiHash: invalid input, string awaited")
		}

		inSeqNo++
		wg.Add(1)
		go handleBatch(strVal, out, wg, inSeqNo, &outSeqNo)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	results := make([]string, 0, 6)
	for input := range in {
		results = append(results, input.(string))
	}

	sort.Slice(results, func(i, j int) bool { return results[i] < results[j] })
	out <- strings.Join(results, "_")
}
