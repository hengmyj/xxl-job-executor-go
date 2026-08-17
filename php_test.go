package xxl

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func phpBinOrSkip(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("php")
	if err != nil {
		t.Skip("php not installed")
	}
	return bin
}

func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := ioutil.TempDir("", "xxl-php-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func TestRunPHPSuccess(t *testing.T) {
	bin := phpBinOrSkip(t)
	out, err := runPHPJob(context.Background(), bin, tempDir(t), &RunReq{
		JobID:          1,
		GlueSource:     "<?php echo 'hello';",
		ExecutorParams: "arg1",
	})
	if err != nil {
		t.Fatalf("unexpected err=%v out=%s", err, out)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("out=%s", out)
	}
}

func TestRunPHPFatalStopsTask(t *testing.T) {
	bin := phpBinOrSkip(t)
	out, err := runPHPJob(context.Background(), bin, tempDir(t), &RunReq{
		JobID:      2,
		GlueSource: "<?php undefined_function();",
	})
	if err == nil {
		t.Fatalf("fatal should fail, out=%s", out)
	}
}

func TestRunPHPParseErrorStopsTask(t *testing.T) {
	bin := phpBinOrSkip(t)
	out, err := runPHPJob(context.Background(), bin, tempDir(t), &RunReq{
		JobID:      3,
		GlueSource: "<?php echo 'oops'",
	})
	if err == nil {
		t.Fatalf("parse error should fail, out=%s", out)
	}
}

func TestRunPHPExceptionStopsTask(t *testing.T) {
	bin := phpBinOrSkip(t)
	out, err := runPHPJob(context.Background(), bin, tempDir(t), &RunReq{
		JobID:      4,
		GlueSource: "<?php throw new Exception('job failed');",
	})
	if err == nil {
		t.Fatalf("exception should fail, out=%s", out)
	}
}

func TestRunPHPWarningStopsTask(t *testing.T) {
	bin := phpBinOrSkip(t)
	out, err := runPHPJob(context.Background(), bin, tempDir(t), &RunReq{
		JobID:      5,
		GlueSource: "<?php $a = $undefined_var; echo 'continued';",
	})
	if err == nil {
		t.Fatalf("warning should fail so the job stops, out=%s", out)
	}
}

func TestRunPHPUserErrorStopsTask(t *testing.T) {
	bin := phpBinOrSkip(t)
	out, err := runPHPJob(context.Background(), bin, tempDir(t), &RunReq{
		JobID:      6,
		GlueSource: "<?php trigger_error('boom', E_USER_ERROR); echo 'continued';",
	})
	if err == nil {
		t.Fatalf("user error should fail, out=%s", out)
	}
}

func TestRunPHPNonZeroExitStopsTask(t *testing.T) {
	bin := phpBinOrSkip(t)
	out, err := runPHPJob(context.Background(), bin, tempDir(t), &RunReq{
		JobID:      7,
		GlueSource: "<?php fwrite(STDERR, 'biz error'); exit(1);",
	})
	if err == nil {
		t.Fatalf("exit(1) should fail, out=%s", out)
	}
}

func TestRunPHPNoticeDoesNotFail(t *testing.T) {
	bin := phpBinOrSkip(t)
	_, err := runPHPJob(context.Background(), bin, tempDir(t), &RunReq{
		JobID:      8,
		GlueSource: "<?php $a = []; echo 'ok';",
	})
	if err != nil {
		t.Fatalf("plain script should succeed: %v", err)
	}
}

func TestRunPHPCancelStopsProcess(t *testing.T) {
	bin := phpBinOrSkip(t)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	out, err := runPHPJob(ctx, bin, tempDir(t), &RunReq{
		JobID:      9,
		GlueSource: "<?php sleep(10); echo 'should not finish';",
	})
	if err == nil {
		t.Fatalf("timeout should stop php, out=%s", out)
	}
}

func TestRunPHPEmptySource(t *testing.T) {
	_, err := runPHPJob(context.Background(), "php", tempDir(t), &RunReq{JobID: 10})
	if err == nil {
		t.Fatal("empty source should fail")
	}
}

func TestNewRunTaskPHP(t *testing.T) {
	e := &executor{opts: Options{PHPBin: "php"}}
	e.regList = &taskList{data: make(map[string]*Task)}
	tk, err := e.newRunTask(&RunReq{GlueType: GlueTypePHP, GlueSource: "<?php echo 1;"})
	if err != nil {
		t.Fatal(err)
	}
	if tk == nil || tk.fn == nil {
		t.Fatal("php glue task should have handler")
	}
	_, err = e.newRunTask(&RunReq{GlueType: GlueTypePHP})
	if err == nil {
		t.Fatal("empty php source should fail before run")
	}
	_, err = e.newRunTask(&RunReq{GlueType: GlueTypeBean, ExecutorHandler: "missing"})
	if err == nil {
		t.Fatal("unregistered bean should fail")
	}
}

func TestPHPTaskRunCallbackFailure(t *testing.T) {
	bin := phpBinOrSkip(t)
	ctx, cancel := context.WithCancel(context.Background())
	e := &executor{opts: Options{PHPBin: bin, LogDir: tempDir(t)}}
	tk := &Task{
		Id:     11,
		Name:   GlueTypePHP,
		Ext:    ctx,
		Cancel: cancel,
		Param: &RunReq{
			JobID:      11,
			GlueType:   GlueTypePHP,
			GlueSource: "<?php throw new Exception('from glue');",
		},
		fn:  e.runPHP,
		log: &nopLogger{},
	}
	var code int64
	tk.Run(func(c int64, m string) {
		code = c
	})
	if code != FailureCode {
		t.Fatalf("php glue error should callback 500, code=%d", code)
	}
}
