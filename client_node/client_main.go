/* Client front-end application */
package main

import (
	"fmt"
	"log"
	"os"

	client_utils "github.com/piercirocaliandro/sdcc-edge-computing/client_utils"
	pb_client "github.com/piercirocaliandro/sdcc-edge-computing/pb_storage"
)

// Client main function for the front-end
func main() {
	registerPort := "1234"
	if len(os.Args) < 2 {
		log.Fatalf("Please specify the register IP address")
	}
	registerAddr := os.Args[1]
	conn := client_utils.SetUpConnection(registerAddr, registerPort)
	defer conn.Close()

	client2 := pb_client.NewStorageClient(conn)
	fmt.Println("\t\t\t\t Welcome!")
	for {
		fmt.Println("Choose an operation to perform:")
		fmt.Println("1) Upload a new element")
		fmt.Println("2) Retrieve an existing element")
		fmt.Println("3) Delete an existing value")
		fmt.Println("4) Add a value to an existing one")
		fmt.Println("5) Exit")

		var decision int
		fmt.Scanln(&decision)
		if decision == 5 {
			fmt.Println("Bye bye !")
			os.Exit(1)
		}

		client_utils.AtLestOnceRetx(decision, &client2, &pb_client.StorageRecordMessage{
			Key:   "",
			Value: []byte("")}, false)
	}
}
