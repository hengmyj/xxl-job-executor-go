package xxl

import (
	"context"
	"fmt"
	"runtime/debug"
)

// TaskFunc 任务执行函数。
// 返回的 error 不为 nil 时，执行器会以 handleCode=500 回调调度中心，任务判定失败并停止。
// 仅返回 msg 且 error 为 nil 时视为成功（handleCode=200）。
type TaskFunc func(cxt context.Context, param *RunReq) (msg string, err error)

// Task 任务
type Task struct {
	Id        int64
	Name      string
	Ext       context.Context
	Param     *RunReq
	fn        TaskFunc
	Cancel    context.CancelFunc
	StartTime int64
	EndTime   int64
	//日志
	log Logger
}

// Run 运行任务。PHP/脚本内部报错必须通过 error 或非 0 退出码传到这里，才能失败回调并停止任务。
func (t *Task) Run(callback func(code int64, msg string)) {
	defer func() {
		if t.Cancel != nil {
			t.Cancel()
		}
	}()
	defer func() {
		if err := recover(); err != nil {
			if t.log != nil {
				t.log.Info(t.Info()+" panic: %v", err)
			}
			debug.PrintStack() //堆栈跟踪
			callback(FailureCode, "task panic:"+fmt.Sprintf("%v", err))
		}
	}()
	if t.fn == nil {
		callback(FailureCode, "task handler is nil")
		return
	}
	msg, err := t.fn(t.Ext, t.Param)
	if err == nil && t.Ext != nil && t.Ext.Err() != nil {
		err = t.Ext.Err()
	}
	if err != nil {
		if msg != "" {
			callback(FailureCode, msg+": "+err.Error())
			return
		}
		callback(FailureCode, err.Error())
		return
	}
	callback(SuccessCode, msg)
}

// Info 任务信息
func (t *Task) Info() string {
	return "任务ID[" + Int64ToStr(t.Id) + "]任务名称[" + t.Name + "]参数：" + t.Param.ExecutorParams
}
