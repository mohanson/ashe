package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/mohanson/gracefulexit"
	"github.com/mohanson/res"
)

const help = `usage: ashe <command> [<args>]

where <command> is one of:
    server, init, kill, reload, list, logs

run 'ashe <command> -h' for more information on a command.`

var (
	pathDataRoot       string
	pathLogs           string
	pathUnixSocketFile string
)

func showProcessStatus(p *ProcessStatus) string {
	return fmt.Sprintf("%-13s %5d %s", p.ProcessConfig.Name, p.Pid, p.ProcessConfig)
}

func mainServer() {
	go func() {
		if err := listen(pathDataRoot); err != nil {
			log.Panicln(err)
		}
	}()
	gracefulexit.Wait()
	client := mainClient()
	l, err := client.list()
	if err != nil {
		log.Panicln(err)
	}
	for _, e := range l {
		if err := client.kill(e.ProcessConfig.Name); err != nil {
			log.Panicln(err)
		}
		log.Println("kill", e.ProcessConfig.Name)
	}
	if err := os.Remove(pathUnixSocketFile); err != nil {
		log.Panicln(err)
	}
	log.Println("remove", pathUnixSocketFile)
	return
}

func mainClient() *client {
	c, err := dial(pathDataRoot)
	if err != nil {
		log.Panicln(err)
	}
	return c
}

func Dash(s []string) ([]string, []string) {
	i := 0
	for ; i < len(s); i++ {
		if s[i] == "--" {
			break
		}
	}
	if i == len(s) {
		return s[:i], []string{}
	}
	return s[:i], s[i+1:]
}

func mainInit() {
	flName := flag.String("n", "temp", "Name")
	lsep, rsep := Dash(os.Args[2:])
	flag.CommandLine.Parse(lsep)
	o := &ProcessConfig{}
	o.Name = *flName
	o.Command = rsep
	o.Dir, _ = os.Getwd()
	o.Env = os.Environ()
	client := mainClient()
	r, err := client.init(o)
	if err != nil {
		log.Panicln(err)
	}
	fmt.Println(showProcessStatus(r))
}

func mainKill() {
	client := mainClient()
	for _, e := range os.Args[2:] {
		if err := client.kill(e); err != nil {
			log.Panicln(err)
		}
	}
}

func mainReload() {
	client := mainClient()
	d, err := client.info(os.Args[2])
	if err != nil {
		log.Panicln(err)
	}
	o := d.ProcessConfig
	if err := client.kill(o.Name); err != nil {
		log.Panicln(err)
	}
	r, err := client.init(o)
	if err != nil {
		log.Panicln(err)
	}
	fmt.Println(r)
}

func mainList() {
	client := mainClient()
	l, err := client.list()
	if err != nil {
		log.Panicln(err)
	}
	for _, docker := range l {
		fmt.Println(showProcessStatus(docker))
	}
}

func mainLogs() {
	var name string
	if len(os.Args) == 2 {
		name = "temp"
	} else {
		name = os.Args[2]
	}
	file := filepath.Join(pathLogs, name)
	cmd := exec.Command("tail", "-n", "32", "-f", file)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	go cmd.Run()
	client := mainClient()
	for {
		time.Sleep(time.Second)
		if _, err := client.info(name); err != nil {
			cmd.Process.Kill()
			break
		}
	}
}

func main() {
	res.BaseExec()
	res.Make("logs")
	pathDataRoot = res.Path("")
	pathLogs = res.Path("logs")
	pathUnixSocketFile = res.Path("ashe.sock")

	if len(os.Args) == 1 {
		fmt.Println(help)
		os.Exit(0)
	}
	if os.Args[1] == "server" {
		mainServer()
		return
	}
	if os.Args[1] == "init" {
		mainInit()
		return
	}
	if os.Args[1] == "kill" {
		mainKill()
		return
	}
	if os.Args[1] == "reload" {
		mainReload()
		return
	}
	if os.Args[1] == "list" {
		mainList()
		return
	}
	if os.Args[1] == "logs" {
		mainLogs()
		return
	}
	fmt.Println(help)
}
