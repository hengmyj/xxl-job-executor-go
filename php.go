package xxl

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const phpErrorTrapFileName = "xxl-job-php-error-trap.php"

// 将 PHP 报错转为非 0 退出，保证执行器能捕捉并停止任务。
// Fatal/Parse/Warning/User Error 以及未捕获异常都会让进程失败。
const phpErrorTrapSource = `<?php
if (defined('XXL_JOB_PHP_ERROR_TRAP')) {
    return;
}
define('XXL_JOB_PHP_ERROR_TRAP', true);

error_reporting(E_ALL);
ini_set('display_errors', '1');
ini_set('display_startup_errors', '1');

set_error_handler(function ($severity, $message, $file, $line) {
    if (!(error_reporting() & $severity)) {
        return false;
    }
    $fail = array(
        E_ERROR,
        E_PARSE,
        E_CORE_ERROR,
        E_COMPILE_ERROR,
        E_USER_ERROR,
        E_RECOVERABLE_ERROR,
        E_WARNING,
        E_USER_WARNING,
        E_CORE_WARNING,
        E_COMPILE_WARNING,
    );
    if (in_array($severity, $fail, true)) {
        throw new ErrorException($message, 0, $severity, $file, $line);
    }
    return false;
});

set_exception_handler(function ($e) {
    fwrite(STDERR, sprintf(
        "PHP %s: %s in %s:%d\n",
        get_class($e),
        $e->getMessage(),
        $e->getFile(),
        $e->getLine()
    ));
    exit(1);
});

register_shutdown_function(function () {
    $err = error_get_last();
    if ($err === null) {
        return;
    }
    $fatal = array(E_ERROR, E_PARSE, E_CORE_ERROR, E_COMPILE_ERROR, E_USER_ERROR);
    if (in_array($err['type'], $fatal, true)) {
        fwrite(STDERR, sprintf(
            "PHP Fatal error: %s in %s:%d\n",
            $err['message'],
            $err['file'],
            $err['line']
        ));
    }
});
`

var (
	phpFailOutput = regexp.MustCompile(`(?i)(Fatal error|Parse error|Uncaught |ErrorException:|PHP (Fatal|Warning|Error))`)
	phpTrapOnce   sync.Once
	phpTrapPath   string
	phpTrapErr    error
)

func isPHPGlue(glueType string) bool {
	switch strings.ToUpper(strings.TrimSpace(glueType)) {
	case GlueTypePHP, "GLUE(PHP)":
		return true
	default:
		return false
	}
}

func (e *executor) runPHP(ctx context.Context, param *RunReq) (string, error) {
	bin := e.opts.PHPBin
	if bin == "" {
		bin = DefaultPHPBin
	}
	return runPHPJob(ctx, bin, e.opts.LogDir, param)
}

// RunPHP 执行 GLUE PHP 源码。PHP 内部报错、警告、未捕获异常或非 0 退出码都会返回 error，
// 调用方应把 error 回传给执行器，从而以失败结束任务。
func RunPHP(ctx context.Context, phpBin string, param *RunReq) (string, error) {
	return runPHPJob(ctx, phpBin, "", param)
}

func runPHPJob(ctx context.Context, phpBin, logDir string, param *RunReq) (string, error) {
	if param == nil {
		return "", fmt.Errorf("php task param is nil")
	}
	if strings.TrimSpace(param.GlueSource) == "" {
		return "", fmt.Errorf("php glue source is empty")
	}
	if phpBin == "" {
		phpBin = DefaultPHPBin
	}
	if ctx == nil {
		ctx = context.Background()
	}

	scriptFile, err := writePHPGlueFile(logDir, param)
	if err != nil {
		return "", err
	}
	trapFile, err := ensurePHPErrorTrap()
	if err != nil {
		return "", fmt.Errorf("write php error trap: %w", err)
	}

	args := []string{
		"-d", "display_errors=1",
		"-d", "display_startup_errors=1",
		"-d", "error_reporting=-1",
		"-d", "auto_prepend_file=" + trapFile,
		scriptFile,
		param.ExecutorParams,
		Int64ToStr(param.BroadcastIndex),
		Int64ToStr(param.BroadcastTotal),
	}
	cmd := exec.CommandContext(ctx, phpBin, args...)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	if ctx.Err() != nil {
		return output, fmt.Errorf("php task stopped: %w", ctx.Err())
	}
	if err != nil {
		if output != "" {
			return output, fmt.Errorf("php task failed: %w", err)
		}
		return "", fmt.Errorf("php task failed: %w", err)
	}
	if phpFailOutput.MatchString(output) {
		return output, fmt.Errorf("php runtime error captured")
	}
	return output, nil
}

func writePHPGlueFile(logDir string, param *RunReq) (string, error) {
	if logDir == "" {
		logDir = filepath.Join(os.TempDir(), "xxl-job")
	}
	dir := filepath.Join(logDir, "gluesource")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create php glue dir: %w", err)
	}
	name := fmt.Sprintf("%d_%d.php", param.JobID, param.GlueUpdatetime)
	path := filepath.Join(dir, name)
	if err := ioutil.WriteFile(path, []byte(param.GlueSource), 0644); err != nil {
		return "", fmt.Errorf("write php glue file: %w", err)
	}
	return path, nil
}

func ensurePHPErrorTrap() (string, error) {
	phpTrapOnce.Do(func() {
		phpTrapPath = filepath.Join(os.TempDir(), phpErrorTrapFileName)
		phpTrapErr = ioutil.WriteFile(phpTrapPath, []byte(phpErrorTrapSource), 0644)
	})
	return phpTrapPath, phpTrapErr
}
