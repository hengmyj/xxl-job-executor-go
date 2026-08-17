package xxl

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type nopLogger struct{}

func (l *nopLogger) Info(format string, a ...interface{})  {}
func (l *nopLogger) Error(format string, a ...interface{}) {}

func runTask(fn TaskFunc) (code int64, msg string) {
	ctx, cancel := context.WithCancel(context.Background())
	t := &Task{
		Id:     1,
		Name:   "test",
		Ext:    ctx,
		Cancel: cancel,
		Param:  &RunReq{ExecutorParams: "p"},
		fn:     fn,
		log:    &nopLogger{},
	}
	t.Run(func(c int64, m string) {
		code, msg = c, m
	})
	return
}

func TestTaskRunSuccess(t *testing.T) {
	code, msg := runTask(func(ctx context.Context, param *RunReq) (string, error) {
		return "ok", nil
	})
	if code != SuccessCode {
		t.Fatalf("code=%d msg=%s", code, msg)
	}
	if msg != "ok" {
		t.Fatalf("msg=%s", msg)
	}
}

func TestTaskRunErrorStopsJob(t *testing.T) {
	code, msg := runTask(func(ctx context.Context, param *RunReq) (string, error) {
		return "php boom", errors.New("exit status 1")
	})
	if code != FailureCode {
		t.Fatalf("want failure, got code=%d msg=%s", code, msg)
	}
	if msg == "" {
		t.Fatal("expected failure message")
	}
}

func TestTaskRunPanicStopsJob(t *testing.T) {
	code, msg := runTask(func(ctx context.Context, param *RunReq) (string, error) {
		panic("php fatal")
	})
	if code != FailureCode {
		t.Fatalf("want failure, got code=%d msg=%s", code, msg)
	}
	if msg == "" {
		t.Fatal("expected panic message")
	}
}

func TestTaskRunCancelStopsJob(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tTask := &Task{
		Id:     2,
		Name:   "cancel",
		Ext:    ctx,
		Cancel: cancel,
		Param:  &RunReq{},
		fn: func(ctx context.Context, param *RunReq) (string, error) {
			return "still running", nil
		},
		log: &nopLogger{},
	}
	var code int64
	tTask.Run(func(c int64, m string) {
		code = c
	})
	if code != FailureCode {
		t.Fatalf("canceled task should fail, code=%d", code)
	}
}

func TestTaskRunTimeoutStopsJob(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	tTask := &Task{
		Id:     3,
		Name:   "timeout",
		Ext:    ctx,
		Cancel: cancel,
		Param:  &RunReq{},
		fn: func(ctx context.Context, param *RunReq) (string, error) {
			<-ctx.Done()
			return "", ctx.Err()
		},
		log: &nopLogger{},
	}
	var code int64
	var msg string
	tTask.Run(func(c int64, m string) {
		code, msg = c, m
	})
	if code != FailureCode {
		t.Fatalf("timeout should fail, code=%d msg=%s", code, msg)
	}
}

func TestTaskRunNilHandler(t *testing.T) {
	code, _ := runTask(nil)
	if code != FailureCode {
		t.Fatalf("nil handler should fail, code=%d", code)
	}
}

func TestIsPHPGlue(t *testing.T) {
	if !isPHPGlue("GLUE_PHP") || !isPHPGlue("GLUE(PHP)") || !isPHPGlue("glue_php") {
		t.Fatal("php glue types should match")
	}
	if isPHPGlue("BEAN") || isPHPGlue("") {
		t.Fatal("non-php glue should not match")
	}
}

func TestTaskInfo(t *testing.T) {
	tk := &Task{Id: 9, Name: "n", Param: &RunReq{ExecutorParams: "x"}}
	if got := tk.Info(); got == "" {
		t.Fatal("empty info")
	}
	_ = fmt.Sprintf("%s", tk.Info())
}
