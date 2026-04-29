//go:build windows

package cmd

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"time"
)

func hideConsoleAfterCountdown() {
	writeCountdown(os.Stdout, time.Sleep)
	freeConsole := syscall.NewLazyDLL("kernel32.dll").NewProc("FreeConsole")
	freeConsole.Call()
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
