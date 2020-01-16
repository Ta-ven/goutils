package main

import (
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

const maxGoroutine = 5

func main() {
	var wg sync.WaitGroup
	wg.Add(maxGoroutine)
	p := &sync.Pool{New: createConnection}
	for query := 0; query < maxGoroutine; query++ {
		go func(q int) {
			dbQuery(q, p)
			wg.Done()
		}(query)
	}
	wg.Wait()
}

func dbQuery(query int, pool *sync.Pool) {
	conn := pool.Get().(*dbConnection)
	defer pool.Put(conn)
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	log.Printf("第%d个查询，使用的是ID为%d的数据库连接", query, conn.ID)
}

type dbConnection struct {
	ID int32
}

func (db *dbConnection) Close() error {
	log.Println("关闭连接", db.ID)
	return nil
}

var idCounter int32

func createConnection() interface{} {
	id := atomic.AddInt32(&idCounter, 1)
	return &dbConnection{ID: id}
}
