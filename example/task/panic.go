package task

import (
	"context"
	xxl "github.com/xxl-job/xxl-job-executor-go"
)

func Panic(cxt context.Context, param *xxl.RunReq) (msg string, err error) {
	panic("test panic")
	return
}
