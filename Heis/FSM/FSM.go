package FSM //Where "FSM" is the folder that contains FSM.go

import (
	"../Driver"
	"../Network"
	"../Orders"
	"../Timer"
)

func Function_state_machine() {
	Arrived_at_floor_chan := make(chan bool, 10)
	Order_chan := make(chan bool, 10)
	Set_timeout_chan := make(chan bool, 10)
	Set_timer_chan := make(chan bool, 10)

	go Network.Network_UDP(Order_chan)
	go Network.Network_order_compare_hall_list()
	go Network.Network_cost_function()
	go Orders.Orders_light_tracking()
	go Orders.Orders_mission_complete(Arrived_at_floor_chan, Set_timeout_chan)
	go Orders.Orders_set_cab_order()
	go Orders.Orders_set_shared_hall_order()
	go Driver.Elev_set_current_floor()
	go Driver.Elev_register_button(Order_chan)
	go Driver.Elev_is_idle(Order_chan)
	go Timer.Timer(Set_timeout_chan, Set_timer_chan, Order_chan)

	for {
		select {

		case <-Arrived_at_floor_chan: //Gets triggered when the elevator has arrived at a desired order. Starts the open-door-timer
			Driver.Elev_set_motor_dir(Driver.DIRN_STOP)
			Driver.Elev_set_door_open_lamp(true)
			Set_timer_chan <- true

		case <-Order_chan: //Gets triggered when an order exists. Gets the next direction for the elevator, and sets it
			dir := Orders.Orders_next_direction()
			Driver.Elev_set_motor_dir(dir)

		case <-Set_timeout_chan: //Gets triggered when the open-door-timer has run out. Light is switched off and the elevator can continue
			Driver.Elev_set_motor_dir(Driver.DIRN_STOP)
			Driver.Elev_set_door_open_lamp(false)
		}
	}
}
