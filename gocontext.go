package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	go doSomething(ctx, "【监控1】")
	go doSomething(ctx, "【监控2】")
	go doSomething(ctx, "【监控3】")

	time.Sleep(10 * time.Second)
	fmt.Println("可以了，通知监控停止")
	cancel()
	//为了检测监控过是否停止，如果没有监控输出，就表示停止了
	time.Sleep(5 * time.Second)
}

func doSomething(ctx context.Context, name string) {
	for {
		select {
		case <-ctx.Done():
			log.Println(name, "监控退出， 停止了")
			return
		default:
			log.Println(name, "goroutine监控中...")
			time.Sleep(1 * time.Second)
		}
	}
}
