package model

import (
	"runtime"

	"github.com/patsnapops/noop/log"
)

// 决定了每个任务同时处理的对象数量。
func GetThreadNum() int64 {
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)

	// 计算可用内存
	availMem := mem.Sys - (mem.HeapSys + mem.StackSys)

	// 一个线程使用的1G内存数量
	threadMem := int64(1024 * 1024 * 2)

	// 计算所需线程数
	numThreads := int64(availMem) / threadMem

	// fmt.Printf("Available memory: %v bytes\n", availMem)
	// fmt.Printf("Thread memory: %v bytes\n", threadMem)
	// fmt.Printf("Number of threads: %v\n", numThreads)
	log.Infof("Available memory: %v bytes", availMem)
	log.Infof("Number of threads: %v", numThreads)
	if numThreads < 1 {
		numThreads = 1
	}
	return numThreads
}
