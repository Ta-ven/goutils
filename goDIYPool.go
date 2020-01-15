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

const maxGoroutine = 5

var ErrPoolClosed = errors.New("资源池已经关闭。")

type Pool struct {
	m       sync.Locker
	res     chan io.Closer            // 资源存放通道
	factory func() (io.Closer, error) //资源产生函数
	closed  bool                      //pool关闭标志
}

func NewPool(fn func() (io.Closer, error), size uint) (*Pool, error) {
	if size <= 0 {
		return nil, errors.New("size的值太小了")
	}
	return &Pool{
		res:     make(chan io.Closer, size),
		factory: fn,
	}, nil
}

//创建资源或者在pool获取资源
func (p *Pool) Acquire() (io.Closer, error) {
	select {
	case r, ok := <-p.res:
		log.Println("Acquire:共享资源")
		if !ok {
			return nil, ErrPoolClosed
		}
		return r, nil
	default:
		log.Println("Acquire创建新的资源")
		return p.factory()
	}
}

func (p *Pool) Close() {
	p.m.Lock()
	defer p.m.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	close(p.res) //关闭资源通道
	for r := range p.res {
		r.Close()
	}
}

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
		log.Println("资源通道满了")
		r.Close()
	}
}

func main() {
	var wg sync.WaitGroup
	wg.Add(maxGoroutine)
	var poolres = uint(2)
	pool, err := NewPool(createConnection1, poolres)
	if err != nil {
		log.Println(err)
		return
	}
	for query := 0; query < maxGoroutine; query++ {
		go func(q int) {
			dbQuery1(q, pool)
			wg.Done()
		}(query)
	}
	wg.Wait()
	log.Println("开始关闭资源池")
	pool.Close()
}

//实例
type dbConnection1 struct {
	ID int32
}

func (db *dbConnection1) Close() error {
	log.Println("关闭连接", db.ID)
	return nil
}

var idCounter1 int32

func createConnection1() (io.Closer, error) {
	id := atomic.AddInt32(&idCounter1, 1)
	return &dbConnection1{ID: id}, nil
}

func dbQuery1(query int, pool *Pool) {
	conn, err := pool.Acquire()
	if err != nil {
		log.Println(err)
		return
	}

	defer pool.Release(conn)

	//模拟查询
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
	log.Printf("第%d个查询，使用的是ID为%d的数据库连接", query, conn.(*dbConnection1).ID)
}
