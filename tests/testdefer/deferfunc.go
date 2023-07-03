package main

import (
	"fmt"
	"time"
)

func main() {
	threadNum := make(chan int, 4)
	for i := 0; i < 10; i++ {
		threadNum <- 1
		go func(i int) {
			defer func() {
				<-threadNum
			}()
			if i == 5 {
				return
			}
			time.Sleep(1 * time.Second)
			fmt.Println("hello", i)
		}(i)
	}
	for {
		if len(threadNum) == 0 {
			break
		}
		fmt.Printf("wait %d", len(threadNum))
		time.Sleep(1 * time.Second)
	}
}
