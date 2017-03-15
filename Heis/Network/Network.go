package Network //Where "Network" is the folder that contains Network.go

import (
	"../Driver"
	"../Orders"
	"../bcast"
	"../localip"
	"../peers"
	"flag"
	"fmt"
	"os"
	"time"
)

type Elevator_info struct { //Struct that can be sent over UDP
	Message       string
	IP            string
	Current_floor int
	Direction     int
	Is_idle       bool
}

var received_msg = [Driver.N_FLOORS][Driver.N_BUTTONS]int{ //The "Message" part of the Elevator_info-struct that is received over UDP. Contains all orders
	{0, 0, 0},
	{0, 0, 0},
	{0, 0, 0},
	{0, 0, 0},
}
var received_IP = "0"              //initial value for IPs
var received_current_floor int = 0 //Initial value for current_floor
var received_direction int = 0     //Initial value for direction
var received_is_idle bool = true   //Initial state of "Is_idle"

var num_elevs_online int = 1

var elev_1_ID int = 0 //Initial values
var elev_2_ID int = 0 //Initial values
var elev_3_ID int = 0 //Initial values

var elev_1 = Elevator_info{Message: "0", IP: "000", Current_floor: 0, Direction: 0, Is_idle: false} //Elev_1 will always be the elevator connected to the pc running the program
var elev_2 = Elevator_info{Message: "0", IP: "000", Current_floor: 0, Direction: 0, Is_idle: false}
var elev_3 = Elevator_info{Message: "0", IP: "000", Current_floor: 0, Direction: 0, Is_idle: false}

func Network_UDP(Order_chan chan bool) { // Initializes UDP-connection, transmits and receives data of type Elevator_info //// Is a go-routine
	var id string

	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()

	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}

	peerUpdateCh := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool)

	go peers.Transmitter(15647, id, peerTxEnable)
	go peers.Receiver(15647, peerUpdateCh)

	helloTx := make(chan Elevator_info)
	helloRx := make(chan Elevator_info)

	go bcast.Transmitter(16569, helloTx)
	go bcast.Receiver(16569, helloRx)

	LocalIP, _ := localip.LocalIP()

	go func() {
		for {
			current_floor1 := Driver.Current_floor
			Dir := Driver.IO_read_bit(Driver.MOTORDIR)
			idle := Driver.Elev_is_idle(Order_chan)
			Message := Elevator_info{Message: Network_orders_to_string(), IP: LocalIP, Current_floor: current_floor1, Direction: Dir, Is_idle: idle}
			helloTx <- Message
			time.Sleep(10 * time.Millisecond)
		}
	}()

	for {
		select {
		case a := <-helloRx:
			received_msg = Network_string_to_orders(a.Message)
			received_IP = a.IP
			received_current_floor = a.Current_floor
			received_direction = a.Direction
			received_is_idle = a.Is_idle
			Network_set_ID_from_IP()

			if received_IP == LocalIP { //Ensures the right information is assigned to the right elevator
				elev_1 = a
			} else if (received_IP != LocalIP) && (elev_2.IP == "000") {
				elev_2.IP = received_IP
			} else if received_IP == elev_2.IP {
				elev_2 = a
			} else if (received_IP != LocalIP) && (received_IP != elev_2.IP) {
				elev_3 = a
			}
			if elev_3.IP != "000" {
				num_elevs_online = 3
			} else if elev_2.IP != "000" {
				num_elevs_online = 2
			} else {
				num_elevs_online = 1
			}
		}
	}
}

func Network_set_ID_from_IP() {
	if _, err := fmt.Sscanf(elev_1.IP, "129.241.187.%3d", &elev_1_ID); err == nil {
	}
	if _, err := fmt.Sscanf(elev_2.IP, "129.241.187.%3d", &elev_2_ID); err == nil {
	}
	if _, err := fmt.Sscanf(elev_3.IP, "129.241.187.%3d", &elev_3_ID); err == nil {
	}
}

func Network_cost_function() { //Looks at floor difference, direction and elevator IDs (considering the number of elevators online) //// Is a go-routine
	for {
		time.Sleep(50 * time.Millisecond)

		var elev_sufficient bool = false
		var elev_1_difference int = 0 //Floor difference between current floor for the elevator and the floor an order is placed in
		var elev_2_difference int = 0
		var elev_3_difference int = 0

		for floor := 0; floor < Driver.N_FLOORS; floor++ { //This for-loop takes care of orders above the elevator

			if Driver.Order_shared_hall_list[floor][0] == 1 { //Takes care of "Up"-buttons

				if num_elevs_online == 1 { //Ensures that one elevator can operate by itself without losing orders
					Driver.Order_hall_list[floor][0] = 1

				} else if num_elevs_online == 2 { //Assigns orders effectively between two elevators
					elev_1_difference = floor - elev_1.Current_floor
					elev_2_difference = floor - elev_2.Current_floor
					elev_3_difference = 5 //Ensures that no orders will be assigned to a "non-existent" elevator

					if (elev_1.Direction == 0 || elev_1.Is_idle == true) && elev_1.Current_floor < floor {
						elev_sufficient = true
					}
					if (elev_2.Direction == 0 || elev_2.Is_idle == true) && elev_2.Current_floor < floor {

						if elev_1_difference > elev_2_difference {
							elev_sufficient = false

						} else if elev_1_difference == elev_2_difference {
							if elev_1_ID < elev_2_ID {
								elev_sufficient = false
							}
						}
					}
				}
			}
			if elev_sufficient == true { //If elev_sufficiant == true, it means that this elevator is fit to take the order
				Driver.Order_hall_list[floor][0] = 1
				elev_sufficient = false
			}
			if Driver.Order_shared_hall_list[floor][1] == 1 { //Takes care of "Down"-buttons

				if num_elevs_online == 1 {
					Driver.Order_hall_list[floor][1] = 1

				} else if num_elevs_online == 2 {
					elev_1_difference = floor - elev_1.Current_floor
					elev_2_difference = floor - elev_2.Current_floor
					elev_3_difference = 5

					if (elev_1.Direction == 0 || elev_1.Is_idle == true) && elev_1.Current_floor < floor {
						elev_sufficient = true
					}
					if (elev_2.Direction == 0 || elev_2.Is_idle == true) && elev_2.Current_floor < floor {

						if elev_1_difference > elev_2_difference {
							elev_sufficient = false

						} else if elev_1_difference == elev_2_difference {
							if elev_1_ID < elev_2_ID {
								elev_sufficient = false
							}
						}
					}
				}
			}
			if elev_sufficient == true {
				Driver.Order_hall_list[floor][1] = 1
				elev_sufficient = false

			} else if num_elevs_online == 3 { //Assigns orders between three elevators,"Up"-buttons
				elev_1_difference = floor - elev_1.Current_floor
				elev_2_difference = floor - elev_2.Current_floor
				elev_3_difference = floor - elev_3.Current_floor

				if (elev_1.Direction == 0 || elev_1.Is_idle == true) && elev_1.Current_floor < floor {
					elev_sufficient = true
				}
				if (elev_2.Direction == 0 || elev_2.Is_idle == true) && elev_2.Current_floor < floor {

					if elev_1_difference > elev_2_difference {
						elev_sufficient = false

					} else if elev_1_difference == elev_2_difference {
						if elev_1_ID < elev_2_ID {
							elev_sufficient = false
						}
					}
				}
				if (elev_3.Direction == 0 || elev_3.Is_idle == true) && elev_3.Current_floor < floor {

					if elev_1_difference > elev_3_difference {
						elev_sufficient = false

					} else if elev_1_difference == elev_3_difference {
						if elev_1_ID < elev_3_ID {
							elev_sufficient = false
						}
					}
				}
			}
			if elev_sufficient == true {
				Driver.Order_hall_list[floor][0] = 1
				elev_sufficient = false
			}
			if Driver.Order_shared_hall_list[floor][1] == 1 { //Assigns orders between three elevators,"Down"-buttons

				if num_elevs_online == 3 {
					elev_1_difference = floor - elev_1.Current_floor
					elev_2_difference = floor - elev_2.Current_floor
					elev_3_difference = floor - elev_3.Current_floor

					if (elev_1.Direction == 0 || elev_1.Is_idle == true) && elev_1.Current_floor < floor {
						elev_sufficient = true
					}
					if (elev_2.Direction == 0 || elev_2.Is_idle == true) && elev_2.Current_floor < floor {

						if elev_1_difference > elev_2_difference {
							elev_sufficient = false

						} else if elev_1_difference == elev_2_difference {
							if elev_1_ID < elev_2_ID {
								elev_sufficient = false
							}
						}
					}
					if (elev_3.Direction == 0 || elev_3.Is_idle == true) && elev_3.Current_floor < floor {

						if elev_1_difference > elev_3_difference {
							elev_sufficient = false

						} else if elev_1_difference == elev_3_difference {
							if elev_1_ID < elev_3_ID {
								elev_sufficient = false
							}
						}
					}
				}
			}
			if elev_sufficient == true {
				Driver.Order_hall_list[floor][1] = 1
				elev_sufficient = false
			}
		}
		for floor := Driver.N_FLOORS - 1; floor >= 0; floor-- { //This for-loop takes care of orders below the elevator, if the structure is unclear, see comments for the "Up"-for-loop
			//(Structure difference: iterates from top to bottom, calculates the floor differece differently)
			if Driver.Order_shared_hall_list[floor][0] == 1 {

				if num_elevs_online == 2 {
					elev_1_difference = elev_1.Current_floor - floor
					elev_2_difference = elev_2.Current_floor - floor
					elev_3_difference = 5

					if (elev_1.Direction == 1 || elev_1.Is_idle == true) && elev_1.Current_floor > floor {
						elev_sufficient = true
					}
					if (elev_2.Direction == 1 || elev_2.Is_idle == true) && elev_2.Current_floor > floor {

						if elev_1_difference > elev_2_difference {
							elev_sufficient = false

						} else if elev_1_difference == elev_2_difference {
							if elev_1_ID < elev_2_ID {
								elev_sufficient = false
							}
						}
					}
				}
			}
			if elev_sufficient == true {
				Driver.Order_hall_list[floor][0] = 1
				elev_sufficient = false
			}

			if Driver.Order_shared_hall_list[floor][1] == 1 {

				if num_elevs_online == 2 {
					elev_1_difference = elev_1.Current_floor - floor
					elev_2_difference = elev_2.Current_floor - floor
					elev_3_difference = 5

					if (elev_1.Direction == 1 || elev_1.Is_idle == true) && elev_1.Current_floor > floor {
						elev_sufficient = true
					}
					if (elev_2.Direction == 1 || elev_2.Is_idle == true) && elev_2.Current_floor > floor {

						if elev_1_difference > elev_2_difference {
							elev_sufficient = false

						} else if elev_1_difference == elev_2_difference {
							if elev_1_ID < elev_2_ID {
								elev_sufficient = false
							}
						}
					}
				}
			}
			if elev_sufficient == true {
				Driver.Order_hall_list[floor][1] = 1
				elev_sufficient = false
			}
			if Driver.Order_shared_hall_list[floor][0] == 1 {

				if num_elevs_online == 3 {
					elev_1_difference = elev_1.Current_floor - floor
					elev_2_difference = elev_2.Current_floor - floor
					elev_3_difference = elev_3.Current_floor - floor

					if (elev_1.Direction == 1 || elev_1.Is_idle == true) && elev_1.Current_floor > floor {
						elev_sufficient = true
					}
					if (elev_2.Direction == 1 || elev_2.Is_idle == true) && elev_2.Current_floor > floor {

						if elev_1_difference > elev_2_difference {
							elev_sufficient = false

						} else if elev_1_difference == elev_2_difference {
							if elev_1_ID < elev_2_ID {
								elev_sufficient = false
							}
						}
					}
					if (elev_3.Direction == 1 || elev_3.Is_idle == true) && elev_3.Current_floor > floor {

						if elev_1_difference > elev_3_difference {
							elev_sufficient = false

						} else if elev_1_difference == elev_3_difference {
							if elev_1_ID < elev_3_ID {
								elev_sufficient = false
							}
						}
					}
				}
			}
			if elev_sufficient == true {
				Driver.Order_hall_list[floor][0] = 1
				elev_sufficient = false
			}
			if Driver.Order_shared_hall_list[floor][1] == 1 {

				if num_elevs_online == 3 {
					elev_1_difference = elev_1.Current_floor - floor
					elev_2_difference = elev_2.Current_floor - floor
					elev_3_difference = elev_3.Current_floor - floor

					if (elev_1.Direction == 1 || elev_1.Is_idle == true) && elev_1.Current_floor > floor {
						elev_sufficient = true
					}
					if (elev_2.Direction == 1 || elev_2.Is_idle == true) && elev_2.Current_floor > floor {

						if elev_1_difference > elev_2_difference {
							elev_sufficient = false

						} else if elev_1_difference == elev_2_difference {
							if elev_1_ID < elev_2_ID {
								elev_sufficient = false
							}
						}
					}
					if (elev_3.Direction == 1 || elev_3.Is_idle == true) && elev_3.Current_floor > floor {

						if elev_1_difference > elev_3_difference {
							elev_sufficient = false

						} else if elev_1_difference == elev_3_difference {
							if elev_1_ID < elev_3_ID {
								elev_sufficient = false
							}
						}
					}
				}
			}
			if elev_sufficient == true {
				Driver.Order_hall_list[floor][1] = 1
				elev_sufficient = false
			}
		}
	}
}

func Network_orders_to_string() string { //Converts orders (list) to a string, so it can be transmitted over UDP
	var Orders string = ""

	for floor := 0; floor < Driver.N_FLOORS; floor++ {
		if Driver.Order_shared_hall_list[floor][0] == 1 {
			Orders = Orders + "1"
		} else {
			Orders = Orders + "0"
		}
	}
	for floor := 0; floor < Driver.N_FLOORS; floor++ {
		if Driver.Order_shared_hall_list[floor][1] == 1 {
			Orders = Orders + "1"
		} else {
			Orders = Orders + "0"
		}
	}
	for floor := 0; floor < Driver.N_FLOORS; floor++ {

		if Driver.Order_cab_list[floor] == 1 {
			Orders = Orders + "1"
		} else {
			Orders = Orders + "0"
		}
	}
	return Orders
}

func Network_string_to_orders(Orders1 string) [4][3]int { //Converts a received string of orders to the desired format (list)

	//U = orders button_up | D = orders button_down | C = orders button_command)
	var Orders_list = [Driver.N_FLOORS][Driver.N_BUTTONS]int{
		{0, 0, 0}, //1st floor, U D C
		{0, 0, 0}, //2nd floor, U D C
		{0, 0, 0}, //3rd floor, U D C
		{0, 0, 0}, //4th floor, U D C
	}

	for i := 0; i < 4; i++ {
		if Orders1[i] == byte(49) {
			Orders_list[i][0] = 1
		} else if Orders1[i] == byte(48) {
			Orders_list[i][0] = 0
		} else {
			fmt.Println("Button_Up " + string(i) + " has an illegal value")
		}
	}
	for j := 4; j < 8; j++ {
		if Orders1[j] == byte(49) {
			Orders_list[j-4][1] = 1
		} else if Orders1[j] == byte(48) {
			Orders_list[j-4][1] = 0
		} else {
			fmt.Println("Button_Down " + string(j) + " has an illegal value")
		}
	}
	for k := 8; k < 12; k++ {
		if Orders1[k] == byte(49) {
			Orders_list[k-8][2] = 1
		} else if Orders1[k] == byte(48) {
			Orders_list[k-8][2] = 0
		} else {
			fmt.Println("Button_Command " + string(k) + "has an illegal value")
		}
	}
	return Orders_list
}

func Network_order_compare_hall_list() { //Compares the shared-hall-orders-list with the received list of orders from other elevators. If there are any new orders, or some old
	//ones have been removed, the shared-list is updated //// Is a go-routine
	for {
		counter := 0
		localIP, _ := localip.LocalIP()

		for floor := 0; floor < 4; floor++ {
			if Driver.Order_shared_hall_list[floor][0] != received_msg[floor][0] && (received_IP != localIP) {
				Driver.Order_shared_hall_list[floor][0] = received_msg[floor][0]
				counter++
			}

			if Driver.Order_shared_hall_list[floor][1] != received_msg[floor][1] && (received_IP != localIP) {
				Driver.Order_shared_hall_list[floor][1] = received_msg[floor][1]
				counter++
			}
		}
		if counter != 0 {
			Orders.Orders_set_new_order_var()
		}
		time.Sleep(50 * time.Millisecond)
	}
}
