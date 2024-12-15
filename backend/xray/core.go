package xray

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sync"

	nodeLogger "github.com/m03ed/marzban-node-go/logger"
)

type Core struct {
	executablePath string
	assetsPath     string
	version        string
	process        *exec.Cmd
	restarting     bool
	logsChan       chan string
	cancelFunc     context.CancelFunc
	mu             sync.Mutex
}

func NewXRayCore(executablePath, assetsPath string) (*Core, error) {
	core := &Core{
		executablePath: executablePath,
		assetsPath:     assetsPath,
		logsChan:       make(chan string),
	}

	version, err := core.refreshVersion()
	if err != nil {
		return nil, err
	}
	core.setVersion(version)

	return core, nil
}

func (c *Core) refreshVersion() (string, error) {
	cmd := exec.Command(c.executablePath, "version")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	r := regexp.MustCompile(`^Xray (\d+\.\d+\.\d+)`)
	matches := r.FindStringSubmatch(out.String())
	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", errors.New("could not parse Xray version")
}

func (c *Core) setVersion(version string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.version = version
}

func (c *Core) GetVersion() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.version
}

func (c *Core) Started() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.process == nil || c.process.Process == nil {
		return false
	}
	if c.process.ProcessState == nil {
		return true
	}
	return false
}

func (c *Core) Start(config *Config) error {
	if c.Started() {
		return errors.New("xray is started already")
	}

	logLevel := config.Log.LogLevel
	if logLevel == "none" || logLevel == "error" {
		config.Log.LogLevel = "warning"
	}

	accessFile, errorFile := config.RemoveLogFiles()

	cmd := exec.Command(c.executablePath, "run", "-config", "stdin:")
	cmd.Env = append(os.Environ(), "XRAY_LOCATION_ASSET="+c.assetsPath)

	xrayJson, err := config.ToJSON()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err = nodeLogger.SetLogFile(accessFile, errorFile); err != nil {
		return err
	}

	cmd.Stdin = bytes.NewBufferString(xrayJson)
	err = cmd.Start()
	if err != nil {
		return err
	}
	c.process = cmd

	ctxCore := c.makeContext()
	// Start capturing process logs
	go c.captureProcessLogs(ctxCore, stdout)
	go c.captureProcessLogs(ctxCore, stderr)

	return nil
}

func (c *Core) Stop() {
	if !c.Started() {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.process.Process.Kill()
	c.process = nil

	c.cancelFunc()

	log.Println("xray core stopped")
}

func (c *Core) Restart(config *Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.restarting {
		return errors.New("xray is already restarted")
	}

	c.restarting = true
	defer func() { c.restarting = false }()

	log.Println("restarting Xray core...")
	c.Stop()
	err := c.Start(config)
	if err != nil {
		return err
	}
	return nil
}

func (c *Core) GetLogs() chan string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.logsChan
}

func (c *Core) makeContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cancelFunc = cancel
	return ctx
}
