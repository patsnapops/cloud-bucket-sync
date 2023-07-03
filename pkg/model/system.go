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

// 获取当前的系统资源状态
func GetSystemStatus() *SystemStatus {
	status := &SystemStatus{}
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	status.AvailMem = mem.Sys - (mem.HeapSys + mem.StackSys)
	status.TotalAlloc = mem.TotalAlloc / 1024 / 1024
	status.HeapAlloc = mem.HeapAlloc / 1024 / 1024
	status.HeapSys = mem.HeapSys / 1024 / 1024
	status.StackSys = mem.StackSys / 1024 / 1024
	status.Sys = mem.Sys / 1024 / 1024
	status.NumGoroutine = runtime.NumGoroutine()
	status.NumCPU = runtime.NumCPU()
	return status
}

type SystemStatus struct {
	TotalAlloc   uint64 `json:"total_alloc"` // 已经分配的内存MB
	HeapAlloc    uint64 `json:"heap_alloc"`  // 堆上目前分配的内存MB
	HeapSys      uint64 `json:"heap_sys"`    // 堆上目前系统申请的内存MB
	StackSys     uint64 `json:"stack_sys"`   // 栈上目前系统申请的内存MB
	Sys          uint64 `json:"sys"`         // 系统申请的内存MB
	AvailMem     uint64 `json:"avail_mem"`   // 可用内存 byte
	NumGoroutine int    `json:"num_goroutine"`
	NumCPU       int    `json:"num_cpu"`
}
