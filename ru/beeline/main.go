package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/linkit360/go-operator/ru/beeline/src"
)

func main() {
	c := make(chan os.Signal, 3)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, os.Kill)
	go func() {
		<-c
		src.OnExit()
		os.Exit(1)
	}()

	src.RunServer()
}
