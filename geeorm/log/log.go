package log

import (
	"io"
	"log"
	"os"
	"sync"
)

var (
	//这些 flag 的有效位通常设计成互不重叠。这样按位或之后，不会互相覆盖，每一位都还能保留自己的含义。
	errorLog = log.New(os.Stdout, "\033[31mERROR\033[0m ", log.LstdFlags|log.Lshortfile)
	infoLog  = log.New(os.Stdout, "\033[34mINFO\033[0m ", log.LstdFlags|log.Lshortfile)
	loggers  = []*log.Logger{errorLog, infoLog}
	mu       sync.Mutex
)

var (
	//println是直接输出日志内容，printf则是格式化输出日志内容
	Error  = errorLog.Println
	Info   = infoLog.Println
	Errorf = errorLog.Printf
	Infof  = infoLog.Printf
)

const (
	InfoLevel = iota
	ErrorLevel
	Disabled
)

func SetLevel(level int) {
	mu.Lock()
	defer mu.Unlock()

	for _, logger := range loggers {
		logger.SetOutput(os.Stdout)
	}

	if ErrorLevel < level {
		errorLog.SetOutput(io.Discard) //丢弃日志输出
	}
	if InfoLevel < level {
		infoLog.SetOutput(io.Discard) //丢弃日志输出
	}
}
