/* Client-side API to access the RPC offered by the server
 * In this file, there is the implementation for the client-side logic of all the RPC operation offered by the
 * RPC server, so
 *	- Put: insert a new key:value Record on the server edge node
 * 	- Get : retrieve a key:value Record
 *	- Append: append a value to and existing key
 *	- Del: delete a key:value record
 *
 * This API allows clients to either use the raw RPCs calls or to rely on the 'AtLestOnceRetx' API, which implements
 * an at least once communication semantic
 */
package client_utils

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"strings"
	"time"

	"github.com/piercirocaliandro/sdcc-edge-computing/pb_register"
	pb_client "github.com/piercirocaliandro/sdcc-edge-computing/pb_storage"
	structs "github.com/piercirocaliandro/sdcc-edge-computing/util_structs"
	"google.golang.org/grpc"
)

/* ---------------------------------------- Connection utility functions ------------------------------------------------------- */

/* Set up the connection between client and edge node, passing for the register node
 *
 * @param registerAddr: IP address of the register node
 * @param registerPort: port of the register node
 */
func SetUpConnection(registerAddr string, registerPort string) *grpc.ClientConn {

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())

	conn, err := grpc.Dial(registerAddr+":"+registerPort, opts...) // establish a channel between Client and Register

	if err != nil {
		log.Fatalf("Error in the connection with RegisterServer: %v", err)
	}

	client := pb_register.NewRegisterClient(conn)
	port, err := GetFreePort()
	if err != nil {
		log.Fatalf("Error during GetFreePort(): %v", err)
	}

	serverAddresses, err := connectToRegister(client, port)
	if err != nil {
		log.Fatalf("Error during get information about peers: %v", err)
	}

	fmt.Println("------------------------------ Trying to connect to the server... ------------------------------")
	fmt.Println()

	var closeServerIp string
	var closeServerPort int32
	var tempLatency time.Duration
	var lowerLatency time.Duration
	firstControl := true

	// connect to the StorageServer with less network latency
	for index := range serverAddresses.IpAddr {
		start := time.Now()
		// establish a channel between client and server, to detect latency
		ctx, canc := context.WithTimeout(context.Background(), 2*time.Second) // time out max of 2 seconds
		defer canc()
		conn, err := grpc.DialContext(ctx, serverAddresses.IpAddr[index]+":"+fmt.Sprint(serverAddresses.Port[index]), grpc.WithInsecure())
		if err != nil {
			fmt.Printf("Error while estabishing connection with the server: %v", err)
		}
		conn.Close()
		tempLatency = time.Since(start)
		if firstControl {
			lowerLatency = tempLatency
			closeServerIp = serverAddresses.IpAddr[index]
			closeServerPort = serverAddresses.Port[index]
			firstControl = false
		}
		if tempLatency < lowerLatency {
			lowerLatency = tempLatency
			closeServerIp = serverAddresses.IpAddr[index]
			closeServerPort = serverAddresses.Port[index]
		}
	}

	fmt.Printf("Connection with server: %v:%v\n", closeServerIp, closeServerPort)
	conn, err = grpc.Dial(closeServerIp+":"+fmt.Sprint(closeServerPort), opts...) // establish a channel between client and server
	if err != nil {
		log.Fatalf("Error while estabishing connection with the server: %v", err)
	}

	return conn
}

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

/* Tries to connect to the register to get the list of edge nodes available
 * @param connection: RegisterClient instance
 * @param port: register port number
 */
func connectToRegister(connection pb_register.RegisterClient, port int) (*pb_register.ListOfStorageServerMsg, error) {
	Info := pb_register.ClientMsg{
		IpAddr: getMyIp(),
		Port:   int32(port),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	peers, err := connection.GetCloseStorageServers(ctx, &Info)
	if err != nil {
		log.Fatalf("Error during GetListOfPeers(): %v", err)
	}
	return peers, nil
}

// Get the client IP address
func getMyIp() string {
	var ret string

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalf("Oops: " + err.Error() + "\n")
		os.Exit(1)
	}

	for _, ip := range addrs {
		if ipnet, ok := ip.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if len(ip.String()) <= 16 {
				ipAddr := ip.String()
				ret = strings.Split(ipAddr, "/")[0]
			}
		}
	}
	return ret
}

/* ----------------------------------------- RPC utility functions ------------------------------------------------------------- */

/* Implements the at least once mechanism for message semantic:
 * if the message is sent and an error is received, caused by a fault in the gRPC call and not
 * by another error (e.g: the requested key was not present neither on the edge node or on the cloud)
 * the client keeps sending the RPC unitil at least one time it succeeds
 * @param decision: RPC type
 * @param client: ProtocolBuffer client to call RPC
 *
 * | - @param key
 * | - @param value:
 * | key:value couple to save on the storage
 */
func AtLestOnceRetx(decision int, client *pb_client.StorageClient, storageRecord *pb_client.StorageRecordMessage, auto bool) (*structs.Record, int) {
	var result = 0 // result is for default 0 (RPC succeeded)

	// Try the first RPC call
	record, ret := ServeDecision(decision, *client, storageRecord, auto)
	result = ret
	retx := 0
	for result == -1 { // retry until the RPC succeeds
		fmt.Println("Trying to retx the RPC call")
		record, ret = ServeDecision(decision, *client, storageRecord, auto)

		result = ret
		retx = retx + 1
		sleepTime := time.Duration(math.Pow(3, float64(retx)))
		time.Sleep(sleepTime * 1000)
	}
	return record, result // return the RPC result, so that the client knows how the RPC gone
}

/* Serve a specific client-side request, contacting the server with the specific RPC. The client can either be a real
 * client or an edge node that is contacting another edge node to send an update message for a given key
 *
 * @param decision: the RPC operation type (Get, Put, Append, Delete)
 * @param connection: StorageClient instance, used to call the RPC server side
 * @param clientRecord: instance passed by the edge server node. Nil if the function is called by a client
 * @param path: path to the dir of the edge server node. Nil if the function is called by a client
 * @param auto: if false, tells that the client has to insert the key and value pair
 *
 * TODO: remove the retx set to 5 at most?
 */
func ServeDecision(decision int, connection pb_client.StorageClient,
	clientRecord *pb_client.StorageRecordMessage, auto bool) (*structs.Record, int) {
	var key string
	var value string
	var forward bool = true // set to true by default

	/* Check if the call to this procedure was made by a client or by a server
	 * If the server is sending an update to another "peer", the key value will be pre-set
	 */
	if strings.EqualFold(clientRecord.GetKey(), "") && !auto {
		getDataInput(decision, &key, &value) // get the input data from the user, and check if the value is a file
	} else {
		forward = clientRecord.GetForward()
		key = clientRecord.GetKey()
		value = string(clientRecord.GetValue())
	}

	// prepare to the rpc call
	ctx := context.Background()
	var length = len([]byte(value))

	/* Put and Append RPCs are quite similar, but they need to instantiate a different type of RPC func */
	if decision == 1 || decision == 5 { // Put rpc call
		fmt.Println("Serving a Put...")
		status, err := connection.Put(ctx, &pb_client.StorageRecordMessage{
			Key:             key,
			Value:           []byte(value),
			Opcode:          int32(decision),
			Forward:         forward,
			Length:          int64(length),
			IpSenderOwner:   clientRecord.GetIpSenderOwner(),
			PortSenderOwner: int32(clientRecord.GetPortSenderOwner()),
		})
		var code = status.GetStatus()
		if err != nil {
			if strings.Contains(err.Error(), "Unavailable desc") {
				fmt.Print(err.Error())
			} else if strings.Contains(err.Error(), "DeadlineExceeded desc") {
				return retWithError("Put timeout exceeded"), -1
			}
			fmt.Println(err.Error())
			return retWithError("Error while connecting to the Put rpc call"), int(code)
		}
	} else if decision == 4 { // Append code, but the logic is the same
		fmt.Println("Serving an Append...")
		status, err := connection.Append(ctx, &pb_client.StorageRecordMessage{
			Key:             key,
			Value:           []byte(value),
			Opcode:          int32(decision),
			Forward:         forward,
			Length:          int64(length),
			IpSenderOwner:   clientRecord.GetIpSenderOwner(),
			PortSenderOwner: int32(clientRecord.GetPortSenderOwner()),
		})
		var code = status.GetStatus()
		if code != 0 || err != nil {
			if err != nil {
				if strings.Contains(err.Error(), "Unavailable desc") {
					fmt.Println(err.Error())
				} else if strings.Contains(err.Error(), "DeadlineExceeded desc") {
					return retWithError("Put timeout exceeded"), -1
				}
				return retWithError("Error while serving Append rpc call"), int(code)
			}
		}
	} else if decision == 2 { // Get operation, now is the client that need to receive a stream of Record
		fmt.Println("Serving a Get...")
		stream, err := connection.Get(ctx)
		if err != nil {
			if strings.Contains(err.Error(), "Unavailable desc") {
				fmt.Print(err.Error())
			} else if strings.Contains(err.Error(), "DeadlineExceeded desc") {
				return retWithError("Put timeout exceeded"), -1
			}

			return retWithError("Error while connecting to the Get rpc call"), -1
		}

		err = stream.Send(&pb_client.StorageRecordMessage{ // send the header message
			Key:     key,
			Opcode:  int32(decision),
			Forward: forward,
		})
		if err != nil {
			if strings.Contains(err.Error(), "Unavailable desc") {
				fmt.Print(err.Error())
			} else if strings.Contains(err.Error(), "DeadlineExceeded desc") {
				return retWithError("Put timeout exceeded"), -1
			}

			return retWithError("Error occurred while retrieving values"), -1
		}

		// Set up for receive the values
		getRecord, err := stream.Recv() // this is the record containing the number of values
		if err != nil {
			if strings.Contains(err.Error(), "Unavailable desc") {
				fmt.Print(err.Error())
			} else if strings.Contains(err.Error(), "DeadlineExceeded desc") {
				return retWithError("Put timeout exceeded"), -1
			}

			retWithError(err.Error())
		}
		newSerRecord := &structs.Record{}
		structs.SetKey(newSerRecord, key)
		fmt.Println("Receiving values associated to key:", key)
		if getRecord.GetNvalues() == 0 {
			return retWithError("Error: the requested key was not present on the edge node \n"), -2
		}

		// Iterate until each value is received
		for index := 0; index < int(getRecord.GetNvalues()); index++ {
			getRecord, err := stream.Recv() // receive the value from the RPC
			if err != nil {
				if strings.Contains(err.Error(), "Unavailable desc") {
					fmt.Print(err.Error())
				} else if strings.Contains(err.Error(), "DeadlineExceeded desc") {
					return retWithError("Put timeout exceeded"), -1
				}

				retWithError(err.Error())
			}
			fmt.Printf("Value: %v\n", string(getRecord.GetValue()))
			structs.AppendValue(newSerRecord, string(getRecord.GetValue()))
		}
		fmt.Println("Get successfully completed")
		return newSerRecord, 0
	} else if decision == 3 { // Del operation, just send the value and delete it
		fmt.Println("Serving a Delete...")
		status, err := connection.Del(ctx, &pb_client.StorageRecordMessage{
			Key:             key,
			Opcode:          int32(decision),
			Forward:         forward,
			IpSenderOwner:   clientRecord.GetIpSenderOwner(),
			PortSenderOwner: int32(clientRecord.GetPortSenderOwner()),
		})
		if status.GetStatus() != 0 || err != nil {
			if err != nil {
				if strings.Contains(err.Error(), "Unavailable desc") {
					fmt.Print(err.Error())
				} else if strings.Contains(err.Error(), "DeadlineExceeded desc") {
					return retWithError("Put timeout exceeded"), -1
				}

			}
			return retWithError("Error occurred while serving Del rpc call"), int(status.GetStatus())
		} else {
			fmt.Println(status.GetMessage())
			fmt.Println("Delete successfully completed")
			return &structs.Record{}, int(status.GetStatus())
		}
	}
	return &structs.Record{}, 0
}

/* ------------------------------------------------- Generic utility functions ------------------------------------------------- */

/* Get the data input from the user, returing the key for the new record and the value assosciated.
 *
 * @param opCode: integer indicating the operation type requested from the client
 * @param key: string, the key
 * @param value: string, the value to send
 *
 */
func getDataInput(opCode int, key *string, value *string) {
	fmt.Println("Insert the key: ") // each operation requires a key, so it does not depend on the opCode
	for *key == "" || key == nil {
		fmt.Scanln(key)
		if *key == "" {
			fmt.Println("A key cannot be an empty string.\nPlease, insert a valid key: ") // each operation requires a key, so it does not depend on the opCode
		}
	}
	if opCode == 1 || opCode == 4 { // Put or Append operation, so it is required a value
		fmt.Println("Insert the value: ")
		in := bufio.NewReader(os.Stdin)
		line, err := in.ReadString('\n')
		if err != nil {
			log.Fatalf("Input not valid")
		}
		*value = line
	}
}

/* Return with an error
 * @param err: the string error to print
 */
func retWithError(err string) *structs.Record {
	log.Printf("%s", err)
	errRecord := structs.Record{}
	structs.SetKey(&errRecord, "")
	return &errRecord
}
