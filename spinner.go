package main

import (
	"fmt"
	"time"

	"github.com/fatih/color"
)

type Spinner struct {
	message string
	stop    chan struct{}
}

func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		stop:    make(chan struct{}),
	}
}

func (s *Spinner) Start() {
	go func() {
		for {
			for _, r := range `-\|/` {
				select {
				case <-s.stop:
					return
				default:
					color.Set(color.FgCyan)
					fmt.Printf("\r%s %c", s.message, r)
					color.Unset()
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
	}()
}

func (s *Spinner) Stop() {
	s.stop <- struct{}{}
	fmt.Print("\r")
	for i := 0; i < len(s.message)+2; i++ {
		fmt.Print(" ")
	}
	fmt.Print("\r")
}
