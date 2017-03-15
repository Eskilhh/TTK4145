package Driver //Where "Driver" is the folder that contains IO.go, IO.c, IO.h, Channels.go, Channels.h and Elevator_control
/*
#cgo CFLAGS: -std=c11
#cgo LDFLAGS: -lcomedi -lm
#include "IO.h"
*/
import "C"

func IO_init() int {
	return int(C.io_init())
}

func IO_set_bit(channel int) {
	C.io_set_bit(C.int(channel))
}

func IO_clear_bit(channel int) {
	C.io_clear_bit(C.int(channel))
}

func IO_write_analog(channel int, value int) {
	C.io_write_analog(C.int(channel), C.int(value))
}

func IO_read_bit(channel int) int {
	return int(C.io_read_bit(C.int(channel)))
}

func IO_read_analog(channel int) int {
	return int(C.io_read_analog(C.int(channel)))
}
