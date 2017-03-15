package main

import (
	"./Driver"
	"./FSM"
)

func main() {
	Driver.Elev_init()           //Initializes the elevator
	FSM.Function_state_machine() //Runs the Function-State-Machine
}
