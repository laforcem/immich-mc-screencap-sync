//go:build windows

package cmd

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"time"
)

const swHide = 0

func hideConsoleAfterCountdown() {
	writeCountdown(os.Stdout, time.Sleep)

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getConsoleWindow := kernel32.NewProc("GetConsoleWindow")
	user32 := syscall.NewLazyDLL("user32.dll")
	showWindow := user32.NewProc("ShowWindow")

	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd != 0 {
		showWindow.Call(hwnd, uintptr(swHide))
	}
}

func writeCountdown(w io.Writer, sleep func(time.Duration)) {
	fmt.Fprint(w, "Started in tray. Closing in ")
	for i := 3; i >= 1; i-- {
		if i < 3 {
			fmt.Fprint(w, " ")
		}
		fmt.Fprintf(w, "%d...", i)
		sleep(time.Second)
	}
	fmt.Fprintln(w)
}
