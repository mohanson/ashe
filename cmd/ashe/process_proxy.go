package main

import (
	"os/exec"
	"sort"
	"sync"
	"syscall"
	"time"
)

// ProcessConfig describe how a process should be create.
type ProcessConfig struct {
	// The name is used within client applications that control the processes.
	Name string
	// The command that will be run when this program is started. The command can be either absolute
	// (e.g. /path/to/programname) or relative (e.g. programname). If it is relative, the supervisord's environment
	// $PATH will be searched for the executable. Programs can accept arguments (e.g. /path/to/program foo bar).
	// The command line can use double quotes to group arguments with spaces in them to pass to the program,
	// (e.g. /path/to/program/name -p "foo bar").
	Command []string
	// A file path representing a directory to which supervisord should temporarily chdir before exec'ing the child.
	Dir string
	// A list of key/value pairs in the form KEY=val that will be placed in the supervisord process' environment
	// (and as a result in all of its child process' environments).
	Env []string
}

// ProcessStatus saved information about a child process.
type ProcessStatus struct {
	// Subprocess pid; 0 when not running.
	Pid int
	// ProcessConfig instance.
	ProcessConfig *ProcessConfig
	// Last time the subprocess was started.
	StartTime time.Time
}

// Process is a struct to manage a subprocess created by daemon.
type Process struct {
	// exec.Cmd instance.
	Cmd *exec.Cmd
	// ProcessStatus instance.
	ProcessStatus *ProcessStatus
}

// Struct for managing process in servers.
type ProcessManager struct {
	logger *logger
	hubber *sync.Map
}

func (m *ProcessManager) init(o *ProcessConfig) (*Process, error) {
	if _, b := m.hubber.Load(o.Name); b {
		return nil, ErrNameHasExists
	}
	cmd := exec.Command(o.Command[0], o.Command[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGINT,
	}
	cmd.Dir = o.Dir
	cmd.Env = o.Env
	f := m.logger.open(o.Name)
	cmd.Stdout = f
	cmd.Stderr = f
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	d := &Process{
		Cmd: cmd,
		ProcessStatus: &ProcessStatus{
			ProcessConfig: o,
			StartTime:     time.Now(),
			Pid:           cmd.Process.Pid,
		},
	}
	m.hubber.Store(o.Name, d)
	go func() {
		cmd.Wait()
		m.hubber.Delete(o.Name)
	}()
	return d, nil
}

func (m *ProcessManager) kill(name string) error {
	v, b := m.hubber.Load(name)
	if b == false {
		return ErrNameNotExists
	}
	d := v.(*Process)
	syscall.Kill(-d.Cmd.Process.Pid, syscall.SIGINT)
	for {
		time.Sleep(time.Millisecond * 200)
		if _, b := m.hubber.Load(name); !b {
			break
		}
	}
	return nil
}

func (m *ProcessManager) info(name string) (*Process, error) {
	v, b := m.hubber.Load(name)
	if b == false {
		return nil, ErrNameNotExists
	}
	return v.(*Process), nil
}

func (m *ProcessManager) list() []*Process {
	r := []*Process{}
	m.hubber.Range(func(_ interface{}, v interface{}) bool {
		r = append(r, v.(*Process))
		return true
	})
	sort.Slice(r, func(i int, j int) bool {
		return r[i].ProcessStatus.StartTime.UnixNano() > r[j].ProcessStatus.StartTime.UnixNano()
	})
	return r
}
