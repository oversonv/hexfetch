package render

import (
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

type winsize struct{ Row, Col, X, Y uint16 }

func ioctl(fd uintptr, req uintptr, arg unsafe.Pointer) bool {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(arg))
	return errno == 0
}

func isTTY(fd uintptr) bool {
	var t syscall.Termios
	return ioctl(fd, syscall.TCGETS, unsafe.Pointer(&t))
}

func fdWidth(fd uintptr) int {
	var ws winsize
	if ioctl(fd, syscall.TIOCGWINSZ, unsafe.Pointer(&ws)) {
		return int(ws.Col)
	}
	return 0
}

// TermWidth возвращает ширину терминала или 0, если stdout не tty
// (вывод в пайп/файл не обрезается). Если ядро отдаёт 0 колонок
// (пустой winsize в proot/pty до первого resize), пробуются
// stderr, stdin, /dev/tty и переменная COLUMNS.
func TermWidth() int {
	if !isTTY(os.Stdout.Fd()) {
		return 0
	}
	for _, fd := range []uintptr{os.Stdout.Fd(), os.Stderr.Fd(), os.Stdin.Fd()} {
		if w := fdWidth(fd); w > 0 {
			return w
		}
	}
	if tty, err := os.OpenFile("/dev/tty", os.O_RDONLY|syscall.O_NOCTTY, 0); err == nil {
		defer tty.Close()
		if w := fdWidth(tty.Fd()); w > 0 {
			return w
		}
	}
	if w, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && w > 0 {
		return w
	}
	return 0
}
