package main

import (
	"fmt"
	"log"
	"os"

	xxl "github.com/xxl-job/xxl-job-executor-go"
	"github.com/xxl-job/xxl-job-executor-go/example/task"
)

func main() {
	exec := xxl.NewExecutor(
		xxl.ServerAddr(getenv("XXL_JOB_ADMIN", "http://127.0.0.1/xxl-job-admin")),
		xxl.AccessToken(getenv("XXL_JOB_ACCESS_TOKEN", "")),
		xxl.ExecutorIp(getenv("XXL_EXECUTOR_IP", "127.0.0.1")),
		xxl.ExecutorPort(getenv("XXL_EXECUTOR_PORT", "9999")),
		xxl.RegistryKey(getenv("XXL_REGISTRY_KEY", "golang-jobs")),
		xxl.PHPBin(getenv("XXL_PHP_BIN", "php")),
		xxl.LogDir(getenv("XXL_LOG_DIR", "")),
		xxl.SetLogger(&logger{}),
	)
	exec.Init()
	//设置日志查看handler
	exec.LogHandler(func(req *xxl.LogReq) *xxl.LogRes {
		return &xxl.LogRes{Code: 200, Msg: "", Content: xxl.LogResContent{
			FromLineNum: req.FromLineNum,
			ToLineNum:   2,
			LogContent:  "这个是自定义日志handler",
			IsEnd:       true,
		}}
	})
	//注册任务handler
	exec.RegTask("task.test", task.Test)
	exec.RegTask("task.test2", task.Test2)
	exec.RegTask("task.panic", task.Panic)
	if err := exec.Run(); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// xxl.Logger接口实现
type logger struct{}

func (l *logger) Info(format string, a ...interface{}) {
	fmt.Println(fmt.Sprintf("自定义日志 - "+format, a...))
}

func (l *logger) Error(format string, a ...interface{}) {
	log.Println(fmt.Sprintf("自定义日志 - "+format, a...))
}
