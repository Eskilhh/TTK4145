package Timer //Where "Timer" is the folder that contains Timer.go

import (
	"../Driver"
	"time"
)

func Timer(timeout chan bool, set_timer chan bool, Order_chan chan bool) { //// Is a go-routine
	const door_open_time = 3 * time.Second //When the door is opened, it stays that way for 3 seconds
	timer := time.NewTimer(0)
	timer.Stop()

	for {
		select {
		case <-set_timer: //Timer starts when the elevator has arrived at a desired order
			timer.Reset(door_open_time)
			Driver.Elev_set_motor_dir(Driver.DIRN_STOP)

		case <-timer.C:
			timer.Stop()
			timeout <- true //Times out after 3 seconds
			Order_chan <- true
		}
	}
}
