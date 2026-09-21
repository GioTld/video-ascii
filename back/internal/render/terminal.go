package render

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// EnableRawMode pone stdin en modo raw: sin buffer de línea ni eco de caracteres.
// Devuelve una función restore que DEBE llamarse al terminar (incluso con defer).
func EnableRawMode() (restore func(), err error) {
	fd := int(os.Stdin.Fd())

	var old syscall.Termios
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd), syscall.TCGETS, uintptr(unsafe.Pointer(&old))); errno != 0 {
		return nil, fmt.Errorf("get terminal state: %w", errno)
	}

	raw := old
	// ICANON: desactiva el buffer de línea (leer byte a byte).
	// ECHO: evita que las teclas se impriman en pantalla.
	raw.Lflag &^= syscall.ICANON | syscall.ECHO
	raw.Cc[syscall.VMIN] = 1  // bloquear hasta leer al menos 1 byte
	raw.Cc[syscall.VTIME] = 0 // sin timeout

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd), syscall.TCSETS, uintptr(unsafe.Pointer(&raw))); errno != 0 {
		return nil, fmt.Errorf("set terminal state: %w", errno)
	}

	return func() {
		syscall.Syscall(syscall.SYS_IOCTL, //nolint:errcheck
			uintptr(fd), syscall.TCSETS, uintptr(unsafe.Pointer(&old)))
	}, nil
}
