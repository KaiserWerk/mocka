package mocka

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/KaiserWerk/mocka/assets"
)

type appType string

const (
	consoleType appType = "console"
	webappType  appType = "webapp"
)

// A Service takes care of creating the source file and building the executable. It can start
// the executable as well.
type Service struct {
	appType  appType
	template string
	exePath  string
	ctx      context.Context
	cf       func()

	exitCode      int
	statusCode    int
	statusMessage string
}

// NewConsoleService creates a new *Service containing a console application which always exits
// with the supplied exit code.
func NewConsoleService(exitCode int) *Service {
	s := Service{
		appType:  consoleType,
		exitCode: exitCode,
	}
	s.ctx, s.cf = context.WithCancel(context.Background())
	s.template = strings.ReplaceAll(assets.ConsoleTemplate, "{{exitCode}}", strconv.Itoa(exitCode))
	return &s
}

// NewWebAppService creates a new *Service containing a web application listening on port, returning
// statusCode and statusMessage in every HTTP response.
func NewWebAppService(port, statusCode int, statusMessage string) *Service {
	s := Service{
		appType:       webappType,
		statusCode:    statusCode,
		statusMessage: statusMessage,
	}
	s.ctx, s.cf = context.WithCancel(context.Background())
	s.template = strings.ReplaceAll(assets.WebAppTemplate, "{{statusCode}}", strconv.Itoa(statusCode))
	s.template = strings.ReplaceAll(s.template, "{{statusMessage}}", strconv.Itoa(statusCode))
	s.template = strings.ReplaceAll(s.template, "{{port}}", strconv.Itoa(port))

	return &s
}

// WriteSource writes the source code of the application into w.
func (s *Service) WriteSource(w io.Writer) error {
	_, err := fmt.Fprint(w, s.template)
	return err
}

// CopySource creates a named file containing the source code of the application.
func (s *Service) CopySource(file string) error {
	fh, err := os.Create(file)
	if err != nil {
		return err
	}
	defer fh.Close()

	_, err = fmt.Fprint(fh, s.template)
	return err
}

// Build actually builds the application into an executable.
func (s *Service) Build() error {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "mocka-*")
	if err != nil {
		return err
	}
	source := filepath.Join(tmpDir, "main.go")
	if err = s.CopySource(source); err != nil {
		return err
	}

	target := filepath.Join(tmpDir, "mocka")
	if runtime.GOOS == "windows" {
		target += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", target, source)
	err = cmd.Run()
	if err != nil {
		return err
	}
	s.exePath = target
	return nil
}

// WriteExe copies the executable content into w.
func (s *Service) WriteExe(w io.Writer) error {
	cont, err := os.ReadFile(s.exePath)
	if err != nil {
		return err
	}

	_, err = w.Write(cont)
	return err
}

// CopyExe copies the executable into a file.
func (s *Service) CopyExe(name string) error {
	cont, err := os.ReadFile(s.exePath)
	if err != nil {
		return err
	}

	return os.WriteFile(name, cont, 0666)
}

// GetExePath returns the path to the built executable. If it has not been
// built yet, GetExePath returns an empty string.
func (s *Service) GetExePath() string {
	return s.exePath
}

// Start starts the built executable. If the supplied context is nil, *Service uses
// an internal context which can be cancelled by calling Stop(). Otherwise you have to
// cancel the supplied context yourself.
func (s *Service) Start(ctx context.Context) error {
	useCtx := s.ctx
	if ctx != nil {
		useCtx = ctx
	}

	cmd := exec.CommandContext(useCtx, s.exePath)
	return cmd.Run()
}

// Stop stops the built executable if it has been started with the internal context.
func (s *Service) Stop() {
	s.cf()
}
