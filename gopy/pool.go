package gopy

import (
	"bufio"
	"context"
	"embed"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/jptrs93/goutil/cmdu"
	"github.com/jptrs93/goutil/contextu"
	"github.com/jptrs93/goutil/logu"
	"github.com/vmihailenco/msgpack/v5"
)

// pyLog is a structured log line emitted by the python worker via
// gopyadapter.log as a base64-msgpack stdout line.
type pyLog struct {
	Level   string            `msgpack:"level"`
	Message string            `msgpack:"message"`
	Context map[string]string `msgpack:"context,omitempty"`
}

var DefaultPool *Pool

func InitDefaultPool(scripts embed.FS, executablePath, entryScript string, n int) {
	if DefaultPool != nil {
		panic("InitDefaultPool called more than once")
	}
	DefaultPool = NewPool(context.Background(), scripts, executablePath, entryScript, n)

	// Set up signal handling
	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan,
		os.Interrupt,    // SIGINT (Ctrl+C)
		syscall.SIGTERM, // Termination request
		syscall.SIGHUP,  // Terminal closed
		syscall.SIGQUIT) // Quit from keyboard

	go func() {
		sig := <-signalChan
		// Call Close() when a signal is received
		if DefaultPool != nil {
			DefaultPool.Close()
		}
		// Convert the signal to an exit code
		// Unix convention: signal code + 128
		var exitCode int
		if signum, ok := sig.(syscall.Signal); ok {
			exitCode = int(signum) + 128
		} else {
			exitCode = 1 // Default non-zero exit for unknown signals
		}
		os.Exit(exitCode)
	}()
}

type Pool struct {
	scripts        embed.FS
	executablePath string
	entryScript    string
	workers        []*PythonWrapper
	nextInd        int
	nextIndMu      sync.Mutex
	tempDir        string
	ctx            context.Context
}

func NewPool(ctx context.Context, scripts embed.FS, executablePath, entryScript string, n int) *Pool {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		panic("unable to create temporary directory")
	}

	rootDir := findRootDir(ctx, scripts)
	err = fs.WalkDir(scripts, rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := scripts.ReadFile(path)
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		fullPath := filepath.Join(tempDir, relPath)
		err = os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			return err
		}
		err = os.WriteFile(fullPath, data, 0644)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		panic(fmt.Sprintf("failed to initialise python scripts temporary dir: %v", err))
	}
	p := &Pool{
		scripts:        scripts,
		executablePath: executablePath,
		entryScript:    entryScript,
		workers:        nil,
		tempDir:        tempDir,
		ctx:            ctx,
	}
	for i := 0; i < n; i++ {
		w := NewPythonWrapper(ctx, executablePath, tempDir, entryScript)
		if _, err := w.InitProcess(); err != nil {
			panic(fmt.Sprintf("failed to initialise python process: %v", err))
		}
		p.workers = append(p.workers, w)
	}
	return p
}

func (p *Pool) Close() {
	for _, w := range p.workers {
		w.Close()
	}
	err := os.RemoveAll(p.tempDir)
	if err != nil {
		slog.ErrorContext(p.ctx, fmt.Sprintf("deleting temporary dir %v: %v", p.tempDir, err))
	}
	slog.InfoContext(p.ctx, fmt.Sprintf("deleted temporary dir: %v", p.tempDir))
}

func MustCallDefault[T any](pythonFunctionName string, inputObj any) T {
	res, err := CallPool[T](DefaultPool, pythonFunctionName, inputObj)
	if err != nil {
		panic(err)
	}
	return res
}

func CallDefault[T any](pythonFunctionName string, inputObj any) (T, error) {
	return CallPool[T](DefaultPool, pythonFunctionName, inputObj)
}

func CallPool[T any](w *Pool, pythonFunctionName string, inputObj any) (T, error) {
	w.nextIndMu.Lock()
	ind := w.nextInd
	w.nextInd++
	w.nextIndMu.Unlock()
	worker := w.workers[ind%len(w.workers)]
	return Call[T](worker, pythonFunctionName, inputObj)
}

type PythonWrapper struct {
	executablePath string
	scriptPath     string
	executableDir  string
	ctx            context.Context
	cancelCause    context.CancelCauseFunc
	Com            cmdu.PipeCommunication
	cmd            *exec.Cmd
	mu             sync.Mutex
	parentCtx      context.Context
}

func NewPythonWrapper(ctx context.Context, executablePath, workingDir, scriptPath string) *PythonWrapper {
	return &PythonWrapper{
		executablePath: executablePath,
		scriptPath:     scriptPath,
		executableDir:  workingDir,
		mu:             sync.Mutex{},
		parentCtx:      logu.ExtendLogContext(ctx, "Python", nil),
	}
}

func (w *PythonWrapper) InitProcess() (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cmd != nil {
		if w.ctx.Err() != nil {
			// todo log reason from ctx
			slog.WarnContext(w.parentCtx, fmt.Sprintf("python worker process dead (%v), restarting", context.Cause(w.ctx)))
		} else {
			// existing process alive
			return 0, nil
		}
	}

	com, err := cmdu.NewPipeCommunication()
	if err != nil {
		return 0, fmt.Errorf("failed initialising process communication pipe: %w", err)
	}

	ctx, cancelCauseFunc := context.WithCancelCause(w.parentCtx)

	cmd := exec.Command(w.executablePath, w.scriptPath)
	cmd.Dir = w.executableDir
	cmd.Env = os.Environ()
	slog.DebugContext(ctx, fmt.Sprintf("start worker process: working dir: %v, executable: %v, script: %v", cmd.Dir, w.executablePath, w.scriptPath))
	cmd.ExtraFiles = []*os.File{com.OtherRead, com.OtherWrite}

	stdout, stderr, _, closeFunc, err := cmdu.InitStdPipes(cmd)
	if err != nil {
		cancelCauseFunc(fmt.Errorf("failed initialising fitter process stdout/stderr: %w", err))
		return 0, context.Cause(ctx)
	}
	contextu.OnCancel(ctx, closeFunc)

	go consumeStdout(ctx, stdout)
	go consumeStderr(ctx, stderr)
	if err := cmd.Start(); err != nil {
		cancelCauseFunc(fmt.Errorf("failed to start python process: %w", err))
		return 0, context.Cause(ctx)
	}

	contextu.OnCancel(ctx, func() { _ = cmd.Process.Kill() })

	// handle child process exiting
	go func() {
		err := cmd.Wait()
		if err != nil {
			cancelCauseFunc(fmt.Errorf("exit error from python process: %v", err))
			return
		}
		if cmd.ProcessState.ExitCode() != 0 {
			cancelCauseFunc(fmt.Errorf("python process ended with bad exit code %v", cmd.ProcessState.ExitCode()))
			return
		}
		cancelCauseFunc(nil)
	}()

	slog.InfoContext(ctx, "waiting for python script ready signal")

	buf := []byte("ready")
	readFinished := make(chan struct{})
	go func() {
		initTimeoutCtx, cancel := context.WithTimeout(ctx, time.Second*30)
		defer cancel()
		select {
		case <-readFinished:
			return
		case <-initTimeoutCtx.Done():
			cancelCauseFunc(fmt.Errorf("python script failed to signal itself as ready after 30s"))
		}
	}()
	n, err := com.ThisRead.Read(buf)
	close(readFinished)
	if err != nil {
		cancelCauseFunc(fmt.Errorf("failed to read 'ready' signal from python script: %w", err))
		return 0, context.Cause(ctx)
	} else if n != len(buf) {
		cancelCauseFunc(fmt.Errorf("failed to read 'ready' signal, expected %v bytes but could only read %v", len(buf), n))
		return 0, context.Cause(ctx)
	}

	slog.InfoContext(ctx, "successfully initialised python process")
	w.ctx = ctx
	w.cancelCause = cancelCauseFunc
	w.Com = com
	w.cmd = cmd
	return cmd.Process.Pid, nil
}

func (w *PythonWrapper) Close() {
	w.cancelCause(nil)
}

func Call[T any](w *PythonWrapper, pythonFunctionName string, inputObj any) (T, error) {
	var result T
	if _, err := w.InitProcess(); err != nil {
		return result, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	inputDataBytes, err := msgpack.Marshal(inputObj)

	if err != nil {
		return result, fmt.Errorf("couldn't serialse input data %v", err)
	}
	if err = cmdu.WriteData([]byte(pythonFunctionName), w.Com.ThisWrite); err != nil {
		w.cancelCause(fmt.Errorf("failed writing data to child process: %v", err))
		return result, err
	}
	if err = cmdu.WriteData(inputDataBytes, w.Com.ThisWrite); err != nil {
		w.cancelCause(fmt.Errorf("failed writing data to child process: %v", err))
		return result, err
	}

	var resultData []byte
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

	go func() {
		resultData, err = readResponseData(w.Com.ThisRead)
		cancel()
	}()

	<-ctx.Done()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = fmt.Errorf("python Call timed out: %w", ctx.Err())
		w.cancelCause(err)
		return result, err
	}

	if err != nil {
		var pythonErr *PythonError
		if errors.As(err, &pythonErr) {
			return result, err
		}
		w.cancelCause(fmt.Errorf("failed reading data to child process: %v", err))
		return result, err
	}
	err = msgpack.Unmarshal(resultData, &result)
	if err != nil {
		return result, fmt.Errorf("unmarshalling result from child process: %v", err)
	}
	return result, nil
}

func readResponseData(r io.Reader) ([]byte, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return nil, err
	}

	payloadLen := int32(binary.BigEndian.Uint32(lenBuf))
	if payloadLen == 0 {
		return []byte{}, nil
	}

	isError := payloadLen < 0
	if isError {
		if payloadLen == -1<<31 {
			return nil, fmt.Errorf("invalid response length %d", payloadLen)
		}
		payloadLen = -payloadLen
	}

	payload := make([]byte, int(payloadLen))
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}

	if !isError {
		return payload, nil
	}

	var pythonErr PythonError
	if err := msgpack.Unmarshal(payload, &pythonErr); err != nil {
		return nil, fmt.Errorf("unmarshalling python error payload: %w", err)
	}
	return nil, &pythonErr
}

// findRootDir identifies the first directory of the embedded files
func findRootDir(ctx context.Context, efs embed.FS) string {
	root := "."
	err := fs.WalkDir(efs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != "." {
			root = path
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		slog.WarnContext(ctx, fmt.Sprintf("resolving root dir: %v", err))
	}
	return root
}

func consumeStdout(ctx context.Context, stdout io.ReadCloser) {
	scanner := bufio.NewScanner(stdout)
	// Allow lines up to 1MB; default 64KB buffer would clip large structured
	// log payloads.
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if rec, ok := tryParseLogLine(line); ok {
			logRecord(ctx, rec)
			continue
		}
		slog.InfoContext(ctx, line)
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		slog.DebugContext(ctx, fmt.Sprintf("error consuming stdout: %v", err))
	}
}

func logRecord(ctx context.Context, rec pyLog) {
	args := make([]any, 0, len(rec.Context))
	for k, v := range rec.Context {
		args = append(args, slog.String(k, v))
	}
	switch rec.Level {
	case "DEBUG":
		slog.DebugContext(ctx, rec.Message, args...)
	case "WARNING", "WARN":
		slog.WarnContext(ctx, rec.Message, args...)
	case "ERROR", "CRITICAL":
		slog.ErrorContext(ctx, rec.Message, args...)
	default:
		slog.InfoContext(ctx, rec.Message, args...)
	}
}

// tryParseLogLine attempts to decode a stdout line as base64-msgpack into a
// pyLog. It returns true only if the bytes decode cleanly and the record has
// a recognised level and non-empty message; otherwise the caller should treat
// the line as plain stdout output.
func tryParseLogLine(line string) (pyLog, bool) {
	raw, err := base64.StdEncoding.DecodeString(line)
	if err != nil {
		return pyLog{}, false
	}
	var rec pyLog
	if err := msgpack.Unmarshal(raw, &rec); err != nil {
		return pyLog{}, false
	}
	if rec.Message == "" || !validLogLevel(rec.Level) {
		return pyLog{}, false
	}
	return rec, true
}

func validLogLevel(l string) bool {
	switch l {
	case "DEBUG", "INFO", "WARNING", "WARN", "ERROR", "CRITICAL":
		return true
	}
	return false
}

func consumeStderr(ctx context.Context, stderr io.ReadCloser) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		slog.InfoContext(ctx, scanner.Text())
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		slog.DebugContext(ctx, fmt.Sprintf("error consuming stderr: %v", err))
	}
}
