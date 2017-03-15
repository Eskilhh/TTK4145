package Driver //Where "Driver" is the folder that contains IO.go, IO.c, IO.h, Channels.go, Channels.h and Elevator_control
/*
#cgo CFLAGS: -std=c11
#cgo LDFLAGS: -lcomedi -lm
#include "IO.h"
*/
//import "C"

import (
	"fmt"
	"time"
)

const N_FLOORS int = 4
const N_BUTTONS int = 3
const MOTOR_SPEED int = 2800

var Current_floor int = 0
var Direction int = 0
var New_order_elev bool = false

var Order_cab_list = [N_FLOORS]int{0, 0, 0, 0} //List of cab orders for floor 1, 2, 3, 4, respectivly

var Order_shared_hall_list = [N_FLOORS][N_BUTTONS - 1]int{ //List containing all hall orders from all elevators
	{0, 0}, //Left column is "Up", Right column is "Down" (This is consistent for all hall lists)
	{0, 0}, //Down button at 1st floor and up button in 4th floor will never be set, as they do not exist,
	{0, 0}, //they exist in the list for structure reasons
	{0, 0},
}

var Order_hall_list = [N_FLOORS][N_BUTTONS - 1]int{ //List containing hall orders for the specific elevator
	{0, 0}, //Left column is "Up", Right column is "Down" (This is consistent for all hall lists)
	{0, 0},
	{0, 0},
	{0, 0},
}

type Elev_button_type_t int

const (
	BUTTON_CALL_UP   = 0
	BUTTON_CALL_DOWN = 1
	BUTTON_COMMAND   = 2
)

type Elev_motor_direction_t int

const (
	DIRN_DOWN = -1
	DIRN_STOP = 0
	DIRN_UP   = 1
)

var Lamp_channel_matrix = [N_FLOORS][N_BUTTONS]int{
	{LIGHT_UP1, LIGHT_DOWN1, LIGHT_COMMAND1},
	{LIGHT_UP2, LIGHT_DOWN2, LIGHT_COMMAND2},
	{LIGHT_UP3, LIGHT_DOWN3, LIGHT_COMMAND3},
	{LIGHT_UP4, LIGHT_DOWN4, LIGHT_COMMAND4},
}

var Button_channel_matrix = [N_FLOORS][N_BUTTONS]int{
	{BUTTON_UP1, BUTTON_DOWN1, BUTTON_COMMAND1},
	{BUTTON_UP2, BUTTON_DOWN2, BUTTON_COMMAND2},
	{BUTTON_UP3, BUTTON_DOWN3, BUTTON_COMMAND3},
	{BUTTON_UP4, BUTTON_DOWN4, BUTTON_COMMAND4},
}

func Elev_init() (int, error) { //Initiates the elevator by making it go down to the first floor, and turns all lights off
	var init_success int = IO_init()

	if init_success == 0 {
		return -1, fmt.Errorf("Unable to initialize elevator hardware")
	}
	for floor := 0; floor < N_FLOORS; floor++ {
		if floor != 0 {
			Elev_set_button_lamp(BUTTON_CALL_DOWN, floor, 0)
		}
		if floor != N_FLOORS-1 {
			Elev_set_button_lamp(BUTTON_CALL_UP, floor, 0)
		}
		Elev_set_button_lamp(BUTTON_COMMAND, floor, 0)
	}
	Elev_set_stop_lamp(false)
	Elev_set_door_open_lamp(false)
	for Elev_get_floor_sensor_signal() != 0 {
		Elev_set_motor_dir(DIRN_DOWN)
	}
	Elev_set_motor_dir(DIRN_STOP)
	Elev_set_floor_indicator(0)
	Current_floor = 0
	Direction = 0
	return 0, nil
}

func Elev_set_motor_dir(dir_n Elev_motor_direction_t) {
	if dir_n == 0 {
		IO_write_analog(MOTOR, 0)
	} else if dir_n > 0 {
		IO_clear_bit(MOTORDIR)
		IO_write_analog(MOTOR, MOTOR_SPEED)
	} else if dir_n < 0 {
		IO_set_bit(MOTORDIR)
		IO_write_analog(MOTOR, MOTOR_SPEED)
	}
}

func Elev_set_button_lamp(button Elev_button_type_t, floor int, value int) (int, error) {

	if floor < 0 || floor > N_FLOORS {
		return -1, fmt.Errorf("Floor has an illegal value")
	}
	if button != BUTTON_CALL_UP && button != BUTTON_CALL_DOWN && button != BUTTON_COMMAND {
		return -1, fmt.Errorf("Button has an illegal value")
	}
	if button == BUTTON_CALL_UP && floor == N_FLOORS-1 {
		return -1, fmt.Errorf("Button up from top floor does not exist")
	}
	if button == BUTTON_CALL_DOWN && floor == 0 {
		return -1, fmt.Errorf("Button down from ground floor does not exist")
	}
	if value != 0 {
		IO_set_bit(Lamp_channel_matrix[floor][button])
	} else {
		IO_clear_bit(Lamp_channel_matrix[floor][button])
	}
	return 0, nil
}

func Elev_set_floor_indicator(floor int) (int, error) {

	// Binary encoding. One light must always be on.

	if floor < 0 || floor > N_FLOORS {
		return -1, fmt.Errorf("Floor has an illegal value")
	}
	if floor&0x02 != 0 {
		IO_set_bit(LIGHT_FLOOR_IND1)
	} else {
		IO_clear_bit(LIGHT_FLOOR_IND1)
	}
	if floor&0x01 != 0 {
		IO_set_bit(LIGHT_FLOOR_IND2)
	} else {
		IO_clear_bit(LIGHT_FLOOR_IND2)
	}
	return 0, nil
}

func Elev_set_door_open_lamp(value bool) {
	if value {
		IO_set_bit(LIGHT_DOOR_OPEN)
	} else {
		IO_clear_bit(LIGHT_DOOR_OPEN)
	}
}

func Elev_set_stop_lamp(value bool) {
	if value {
		IO_set_bit(LIGHT_STOP)
	} else {
		IO_clear_bit(LIGHT_STOP)
	}
}

func Elev_get_button_signal(button Elev_button_type_t, floor int) (int, error, int) {

	if floor < 0 || floor > N_FLOORS {
		return -1, fmt.Errorf("Floor has an illegal value"), 0
	}
	if button != BUTTON_CALL_UP && button != BUTTON_CALL_DOWN && button != BUTTON_COMMAND {
		return -1, fmt.Errorf("Button has an illegal value"), 0
	}
	if button == BUTTON_CALL_UP && floor == N_FLOORS-1 {
		return -1, fmt.Errorf("Button up from top floor does not exist"), 0
	}
	if button == BUTTON_CALL_DOWN && floor == 0 {
		return -1, fmt.Errorf("Button down from ground floor does not exist"), 0
	}
	return 0, nil, IO_read_bit(Button_channel_matrix[floor][button])
}

func Elev_get_floor_sensor_signal() int {
	if IO_read_bit(SENSOR_FLOOR1) != 0 {
		return 0
	} else if IO_read_bit(SENSOR_FLOOR2) != 0 {
		return 1
	} else if IO_read_bit(SENSOR_FLOOR3) != 0 {
		return 2
	} else if IO_read_bit(SENSOR_FLOOR4) != 0 {
		return 3
	} else {
		return -1
	}
}

func Elev_check_all_buttons() int {
	for {
		for floor := 0; floor < N_FLOORS; floor++ {
			if _, _, x := Elev_get_button_signal(BUTTON_CALL_UP, floor); x == 1 {
				return Button_channel_matrix[floor][0]
			} else if _, _, x := Elev_get_button_signal(BUTTON_CALL_DOWN, floor); x == 1 {
				return Button_channel_matrix[floor][1]
			} else if _, _, x := Elev_get_button_signal(BUTTON_COMMAND, floor); x == 1 {
				return Button_channel_matrix[floor][2]
			}
		}
	}
	return -1
}

func Elev_register_button(Order_chan chan bool) { //Registers that someone has pushed a button, and sets Order_chan to true //// Is a go-routine
	for {
		for floor := 0; floor < N_FLOORS; floor++ {
			if Elev_check_all_buttons() == Button_channel_matrix[floor][0] {
				if IO_read_bit(LIGHT_DOOR_OPEN) == 0 {
					Order_chan <- true
				}

			} else if Elev_check_all_buttons() == Button_channel_matrix[floor][1] {
				if IO_read_bit(LIGHT_DOOR_OPEN) == 0 {
					Order_chan <- true
				}

			} else if Elev_check_all_buttons() == Button_channel_matrix[floor][2] {
				if IO_read_bit(LIGHT_DOOR_OPEN) == 0 {
					Order_chan <- true
				}
			}

			if New_order_elev == true {
				if IO_read_bit(LIGHT_DOOR_OPEN) == 0 {
					New_order_elev = false
					Order_chan <- true
				}
			}
		}
	}
}

func Elev_is_idle(Order_chan chan bool) bool { //Checks if the elevator is idle, e.g. has no orders assigned to it //// Is a go-routine
	for floor := 0; floor < N_FLOORS; floor++ {
		if Order_cab_list[floor] == 1 || Order_hall_list[floor][0] == 1 || Order_hall_list[floor][0] == 1 {
			if IO_read_bit(LIGHT_DOOR_OPEN) == 0 {
				Order_chan <- true
			}
			return false
		}
	}
	return true
}

func Elev_set_current_floor() { //// Is a go-routine
	for {
		temp := Elev_get_floor_sensor_signal()
		if temp != -1 {
			Current_floor = temp
		}
		time.Sleep(50 * time.Millisecond)
	}
}
