package main
import (
	"os"
	"time"
	"os/signal"
	"sync"
	"fmt"
	"runtime"
	"errors"
)

//超时错误
var syncErrTimeout = errors.New("received timeout")
//操作系统系统中断错误
var syncErrInterrupt = errors.New("received interrupt")

//异步执行任务
type syncRunner struct {
	//操作系统的信号检测
	interrupt chan os.Signal
	//记录执行完成的状态
	complete chan error
	//超时检测
	timeout <-chan time.Time
	//保存所有要执行的任务,顺序执行
	tasks []func(id int) error
	waitGroup sync.WaitGroup
	lock sync.Mutex
	errs chan error
}

//new一个Runner对象
func NewRunner(d time.Duration) *syncRunner {
	return &syncRunner{
		interrupt: make(chan os.Signal, 1),
		complete: make(chan error),
		timeout: time.After(d),
		waitGroup: sync.WaitGroup{},
	}
}

//添加一个任务
func (this *syncRunner) Add(tasks ...func(id int) error) {
	this.tasks = append(this.tasks, tasks...)
}

//启动Runner，监听错误信息
func (this *syncRunner) Start() error {
	//接收操作系统信号
	signal.Notify(this.interrupt, os.Interrupt)
	//并发执行任务
	go func() {
		this.complete <- this.Run()
	}()
	select {
	//返回执行结果
	case err := <-this.complete:
		return err
		//超时返回
	case <-this.timeout:
		return syncErrTimeout
	}
}

//异步执行所有的任务
func (this *syncRunner) Run() error {
	for id, task := range this.tasks {
		if this.gotInterrupt() {
			return syncErrInterrupt
		}
		this.waitGroup.Add(1)
		go func(id int) {
			//执行任务
			err := task(id)
			//加锁保存到结果集中
			this.errs <- err
			this.waitGroup.Done()
		}(id)
	}
	this.waitGroup.Wait()
	close(this.errs)
	return nil
}

//判断是否接收到操作系统中断信号
func (this *syncRunner) gotInterrupt() bool {
	select {
	case <-this.interrupt:
		//停止接收别的信号
		signal.Stop(this.interrupt)
		return true
		//正常执行
	default:
		return false
	}
}

//获取执行完的error
func (this *syncRunner) GetErrs() {
	for i := range this.errs{
		fmt.Println(i)
	}
}

func main() {
	//开启多核
	runtime.GOMAXPROCS(runtime.NumCPU())
	//创建runner对象，设置超时时间
	runner := NewRunner(10 * time.Second)
	//添加运行的任务
	runner.Add(ta(),ta(), ta(), ta(), ta(), ta(), )
	runner.errs = make(chan error, len(runner.tasks))
	fmt.Println("同步执行任务")
	//开始执行任务
	if err := runner.Start(); err != nil {
		switch err {
		case syncErrTimeout:
			fmt.Println("执行超时")
			os.Exit(1)
		case syncErrInterrupt:
			fmt.Println("任务被中断")
			os.Exit(2)
		}
	}
	fmt.Println("执行结束")
	runner.GetErrs()
}

//创建要执行的任务
func ta()func(int)error{
	return func(id int) error{
		fmt.Printf("正在执行%v个任务\n", id)
		//time.Sleep(1 * time.Second)
		return nil
	}
}

