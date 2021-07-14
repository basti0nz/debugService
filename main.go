package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron"
	"github.com/takama/daemon"
)

const (
	name        = "igpu-debug"
	description = "Debug service for igpu"

	workdir    = "/var/log/igpu-debug"
	bufferSize = 64

	// DEV const
	//sourceFile         = "/dev/urandom"
	//cronString         = "*/10 * * * * *"
	//DeleteAfterSeconds = 20

	//PROD const
	sourceFile         = "/sys/kernel/debug/tracing/trace_pipe"
	cronString         = "0 */10 * * * *"
	DeleteAfterSeconds = 1200
)

var quit chan bool
var stdlog, errlog *log.Logger

// Service is the daemon service struct
type Service struct {
	daemon.Daemon
}

func igpuLogRotator() {
	stdlog.Println("Rotator started")
	quit <- true
	go logWorker()

	logFiles, err := os.Open(workdir)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer logFiles.Close()
	files, err := logFiles.Readdir(0)
	if err != nil {
		errlog.Println("Error clean files: ", err)
		return
	}

	time2old := time.Now().Unix() - DeleteAfterSeconds
	for _, v := range files {
		if !v.IsDir() {
			if time2old > v.ModTime().Unix() {
				fullPath := fmt.Sprintf("%s/%s", workdir, v.Name())
				stdlog.Println("Remove file ", fullPath)
				err = os.Remove(fullPath)
				if err != nil {
					errlog.Println("Error remove file: ", fullPath, err)
				}
			}
		}
	}
	stdlog.Println("Rotator stoped")
}

func logWorker() {
	stdlog.Println("logWorker started ")
	source, err := os.Open(sourceFile)
	if err != nil {
		errlog.Println("Error open "+sourceFile+": ", err)
		os.Exit(1)
	}
	defer source.Close()

	fileName := fmt.Sprintf("%s/%s.txt", workdir, time.Now().Format(time.RFC3339))
	dest, err := os.OpenFile(fileName,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		errlog.Println("Error open "+fileName+": ", err)
		os.Exit(1)
	}
	defer dest.Close()

	for {
		select {
		case <-quit:
			stdlog.Println("logWorker stoped ")
			return
		default:
			b := make([]byte, bufferSize)
			_, err := source.Read(b)
			if err == io.EOF {
				break
			} else if err != nil {
				errlog.Println("Error get data from source: ", err)
			} else {
				if _, err := dest.Write(b); err != nil {
					errlog.Println("Error write data to file: ", err)
				}
			}
		}
	}
}

// Manage by daemon commands or run the daemon
func (service *Service) Manage() (string, error) {

	usage := "Usage: " + name + " install | remove | start | stop | status"

	if len(os.Args) > 1 {
		command := os.Args[1]
		switch command {
		case "install":
			return service.Install()
		case "remove":
			return service.Remove()
		case "start":
			return service.Start()
		case "stop":
			return service.Stop()
		case "status":
			return service.Status()
		default:
			return usage, nil
		}
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

	// Create a new cron manager
	c := cron.New()
	// Run makefile every min
	err := c.AddFunc(cronString, igpuLogRotator)
	if err != nil {
		errlog.Println("Error create cron  igpuLogRotator: ", err)
		return "", err
	}
	c.Start()
	go logWorker()
	killSignal := <-interrupt
	stdlog.Println("Got signal:", killSignal)
	return "Service exited", nil
}

func init() {
	stdlog = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	errlog = log.New(os.Stderr, "", log.Ldate|log.Ltime)

}

func main() {
	quit = make(chan bool)
	defer close(quit)

	if _, err := os.Stat(workdir); os.IsNotExist(err) {
		err := os.Mkdir(workdir, os.ModeDir)
		if err != nil {
			errlog.Println("Error: ", err)
			os.Exit(1)
		}
	}

	srv, err := daemon.New(name, description, daemon.SystemDaemon)
	if err != nil {
		errlog.Println("Error: ", err)
		os.Exit(1)
	}
	service := &Service{srv}
	status, err := service.Manage()
	if err != nil {
		errlog.Println(status, "\nError: ", err)
		os.Exit(1)
	}
	fmt.Println(status)
}
