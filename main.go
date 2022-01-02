package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
)

func main() {

	programPath := getProgramPath()
	os.Chdir(programPath)

	pgid := startDebugging()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Panic(err)
	}
	done := make(chan bool)
	go func() {
		for {
			select {
			case evnt, ok := <-watcher.Events:
				if !ok {
					return
				}
				if !strings.HasSuffix(evnt.Name, ".go") {
					continue
				}
				println("restarting..")
				syscall.Kill(-pgid, 15)
				pgid = startDebugging()
				println("..restarted")
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()
	err = watcher.Add(programPath)
	if err != nil {
		log.Fatal(err)
	}
	<-done

}

func startDebugging() int {
	cmd := exec.Command("dlv", "debug", "--headless", "--listen=:2345", "--api-version=2", "--accept-multiclient")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGTERM,
	}

	stdout, _ := cmd.StdoutPipe()
	cmd.Start()

	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanWords)
	go func() {
		for scanner.Scan() {
			m := scanner.Text()
			fmt.Println(m)
		}
	}()
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		panic(err)
	}
	return pgid
}

func getProgramPath() string {
	mydir, err := os.Getwd()
	if err != nil {
		log.Panic(err)
	}
	programPath := filepath.Join(mydir, os.Args[1])
	return programPath
}
