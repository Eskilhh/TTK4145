package Orders //Where "Orders" is the folder that contains Orders.go

import (
	"../Driver"
	"time"
)

var more_orders_upwards bool = false
var more_orders_downwards bool = false

func Orders_mission_complete(Arrived_at_floor_chan chan bool, Set_timeout_chan chan bool) { //Ensures the elevator stops at orders in the right order
	//Whenever an order is complete, the Arrived_at_floor_chan is set //// Is a go-routine

	at_floor := Driver.Elev_get_floor_sensor_signal()
	if at_floor != -1 {
		if Driver.IO_read_bit(Driver.MOTORDIR) == 0 { //Considers stopping when moving upwards
			if Driver.Order_hall_list[at_floor][0] == 1 || Driver.Order_cab_list[at_floor] == 1 { //Always stop if "Up-order" or "Cab order" when moving upwards
				Driver.Order_hall_list[at_floor][0] = 0
				select {
				case <-Set_timeout_chan:
					Driver.Order_cab_list[at_floor] = 0
					Driver.Elev_set_door_open_lamp(false)
				}
				Arrived_at_floor_chan <- true
			}
			if Driver.Order_hall_list[at_floor][1] == 1 { //If down button pressed at current floor when moving upwards, check if more orders above
				if more_orders_upwards == true { //Do not stop if more orders
					for floor := Driver.N_FLOORS; floor > Driver.Current_floor; floor-- {
						if Driver.Order_hall_list[floor][1] == 0 && Driver.Order_hall_list[at_floor][1] == 1 {
							break
						}
					}
				} else {
					Driver.Order_hall_list[at_floor][1] = 0
					select {
					case <-Set_timeout_chan:
						Driver.Order_cab_list[at_floor] = 0
						Driver.Elev_set_door_open_lamp(false)
					}
					Arrived_at_floor_chan <- true
				}
			}
		} else if Driver.IO_read_bit(Driver.MOTORDIR) == 1 { //Considers stopping when moving downwards
			if Driver.Order_hall_list[at_floor][1] == 1 || Driver.Order_cab_list[at_floor] == 1 { //Always stop if "Down-order" or "Cab order" when moving downwards
				Driver.Order_hall_list[at_floor][1] = 0
				select {
				case <-Set_timeout_chan:
					Driver.Order_cab_list[at_floor] = 0
					Driver.Elev_set_door_open_lamp(false)
				}
				Arrived_at_floor_chan <- true
			}
			if Driver.Order_hall_list[at_floor][0] == 1 { //If up button pressed at current floor when moving downwards, check if more orders below
				if more_orders_downwards == true { //Do not stop if more orders
					for floor := 0; floor < Driver.Current_floor; floor++ {
						if Driver.Order_hall_list[floor][0] == 1 && Driver.Order_hall_list[at_floor][0] == 1 {
							break
						}
					}
				} else {
					Driver.Order_hall_list[at_floor][0] = 0
					select {
					case <-Set_timeout_chan:
						Driver.Order_cab_list[at_floor] = 0
						Driver.Elev_set_door_open_lamp(false)
					}
					Arrived_at_floor_chan <- true
				}
			}
		}
	}
}

func Orders_next_direction() Driver.Elev_motor_direction_t { //Ensures that the elevator does not change direction unless no more orders exist current direction, returns direction

	more_orders_upwards = false
	more_orders_downwards = false

	for floor := Driver.Current_floor + 1; floor < Driver.N_FLOORS; floor++ { //Checks if there are more orders above current floor
		if Driver.Order_cab_list[floor] == 1 || Driver.Order_hall_list[floor][0] == 1 || Driver.Order_hall_list[floor][1] == 1 {
			more_orders_upwards = true
			break
		}
	}

	for floor := Driver.Current_floor - 1; floor >= 0; floor-- { //Checks if there are more orders below current floor
		if Driver.Order_cab_list[floor] == 1 || Driver.Order_hall_list[floor][0] == 1 || Driver.Order_hall_list[floor][1] == 1 {
			more_orders_downwards = true
			break
		}
	}
	//Conditions for the returned direction based on direction, orders below and orders above
	if (Driver.IO_read_bit(Driver.MOTORDIR) == 0 && more_orders_upwards == true) || (Driver.IO_read_bit(Driver.MOTORDIR) == 1 && more_orders_upwards == true && more_orders_downwards == false) {
		return Driver.DIRN_UP
	} else if (Driver.IO_read_bit(Driver.MOTORDIR) == 0 && more_orders_upwards == false && more_orders_downwards == true) || (Driver.IO_read_bit(Driver.MOTORDIR) == 1 && more_orders_downwards == true) {
		return Driver.DIRN_DOWN
	}
	return Driver.DIRN_STOP
}

func Orders_set_cab_order() { //// Is a go-routine
	for {
		for floor := 0; floor < Driver.N_FLOORS; floor++ {
			if Driver.Elev_check_all_buttons() == Driver.Button_channel_matrix[floor][2] && floor != Driver.Elev_get_floor_sensor_signal() {
				Driver.Order_cab_list[floor] = 1
			}
		}
	}
}

func Orders_set_shared_hall_order() { //This function adds hall orders from this specific elevator to the list that contains all hall orders from all elevators //// Is a go-routine
	for {
		for floor := 0; floor < Driver.N_FLOORS-1; floor++ {
			if Driver.Elev_check_all_buttons() == Driver.Button_channel_matrix[floor][0] && floor != Driver.Elev_get_floor_sensor_signal() {
				Driver.Order_shared_hall_list[floor][0] = 1

			}
		}
		for floor := 1; floor < Driver.N_FLOORS; floor++ {
			if Driver.Elev_check_all_buttons() == Driver.Button_channel_matrix[floor][1] && floor != Driver.Elev_get_floor_sensor_signal() {
				Driver.Order_shared_hall_list[floor][1] = 1
			}
		}
	}
}

func Orders_set_new_order_var() {
	Driver.New_order_elev = true
}

func Orders_light_tracking() { //Sets and clears lights as needed //// Is a go-routine
	for {
		Driver.Elev_set_floor_indicator(Driver.Elev_get_floor_sensor_signal())

		for floor := 0; floor < Driver.N_FLOORS; floor++ {
			if Driver.Order_shared_hall_list[floor][0] == 1 {
				Driver.Elev_set_button_lamp(Driver.BUTTON_CALL_UP, floor, 1)
			} else {
				Driver.Elev_set_button_lamp(Driver.BUTTON_CALL_UP, floor, 0)
			}
			if Driver.Order_shared_hall_list[floor][1] == 1 {
				Driver.Elev_set_button_lamp(Driver.BUTTON_CALL_DOWN, floor, 1)
			} else {
				Driver.Elev_set_button_lamp(Driver.BUTTON_CALL_DOWN, floor, 0)
			}
			if Driver.Order_cab_list[floor] == 1 {
				Driver.Elev_set_button_lamp(Driver.BUTTON_COMMAND, floor, 1)
			} else {
				Driver.Elev_set_button_lamp(Driver.BUTTON_COMMAND, floor, 0)
			}
		}
	}
	time.Sleep(50 * time.Millisecond)
}
