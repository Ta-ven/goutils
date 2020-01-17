package main

import (
	"errors"
	"io"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

const (
	maxGoroutine = 5
	poolRes      = 2
)

var ErrPoolClosed = errors.New("资源池已经关闭。")

type Pool struct {
	m       sync.Mutex
	res     chan io.Closer
	factory func() (io.Closer, error)
	closed  bool
}

func NewPool(fn func() (io.Closer, error), size uint) (*Pool, error) {
	if size <= 0 {
		return nil, errors.New("数值太小了")
	}
	return &Pool{
		factory: fn,
		res:     make(chan io.Closer, size),
	}, nil
}

//获取资源
func (p *Pool) Acquire() (io.Closer, error) {
	select {
	case r, ok := <-p.res:
		if ok {
			log.Println("从连接池拿到数据")
			return nil, ErrPoolClosed
		}
		return r, nil
	default:
		log.Println("获取新资源")
		return p.factory()
	}
}

//关闭资源
func (p *Pool) Close() {
	p.m.Lock()
	defer p.m.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	close(p.res)
	for r := range p.res {
		r.Close()
	}
}

//回收资源
func (p *Pool) Release(r io.Closer) {
	p.m.Lock()
	defer p.m.Unlock()
	if p.closed {
		r.Close()
		return
	}
	select {
	case p.res <- r:
		log.Println("回收资源")
	default:
		log.Println("池子满了")
		r.Close()
	}
}

func main() {
	var wg sync.WaitGroup
	wg.Add(maxGoroutine)
	pool, err := NewPool(createConn, poolRes)
	if err != nil {
		log.Println(err)
		return
	}
	for i := 0; i < maxGoroutine; i++ {
		go func(i int) {
			dbQuery(i, pool)
			wg.Done()
		}(i)
	}
	wg.Wait()
	log.Println("开始关闭资源池")
	time.Sleep(4 * time.Second)
	dbQuery(1, pool)
	pool.Close()
}

type dbConnection struct {
	ID int32
}

func (db *dbConnection) Close() error {
	log.Println("关闭连接", db.ID)
	return nil
}

var idCounter int32

func createConn() (io.Closer, error) {
	id := atomic.AddInt32(&idCounter, 1)
	return &dbConnection{ID: id}, nil
}

func dbQuery(query int, pool *Pool) {
	conn, err := pool.Acquire()
	if err != nil {
		log.Println(err)
		return
	}
	defer pool.Release(conn)

	//模拟查询
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	log.Printf("第%d个查询，使用的是ID为%d的数据库连接", query, conn.(*dbConnection).ID)
}
