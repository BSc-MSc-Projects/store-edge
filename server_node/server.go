/* This file contains the logic for the edge server node. */
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	client_utils "github.com/piercirocaliandro/sdcc-edge-computing/client_utils"
	"github.com/piercirocaliandro/sdcc-edge-computing/pb_register"
	pb_storage "github.com/piercirocaliandro/sdcc-edge-computing/pb_storage"
	server_utils "github.com/piercirocaliandro/sdcc-edge-computing/server_utils"
	structs "github.com/piercirocaliandro/sdcc-edge-computing/util_structs"
	"google.golang.org/grpc"
)

var peerAddress *pb_register.ListOfStorageServerMsg
var Info pb_register.StorageServerMsg

type storageServer struct {
	pb_storage.UnimplementedStorageServer                  // interface for the grpc server side
	savedValues                           []structs.Record // temporary "storage" for the key-values couples
	folder                                string           // folder to keep small files
	address                               structs.Socket   // the ip address and port of the server
	THRESHOLD_MAX_FILE                    int64            // max size of the file that servers save in local

	lock sync.RWMutex
}

const initialTTL = 600.0             // initial TimeToLive for a resource kept by the edge server
const maxTimeSinceLastUpdate = 500.0 // the time elapsed since the last update of the values ​​of a key

/*--------------------------------------------- RPC functions -----------------------------------------------------------------*/

/* Get RPC call. Retrieve a value for the given key, returns nil if the key does not exists. Server-side streaming RCP call
 *
 * @param ctx: Context
 * @param sr: StorageRecord instance, containing the kay to retrieve
 */
func (s *storageServer) Get(stream pb_storage.Storage_GetServer) error {
	fmt.Println("Received an rpc for a Get operation")

	// Receive the first message from the client, to know which key is requested
	record, err := stream.Recv()
	if err != nil {
		log.Fatalf("Error in get while receiving header")
	}

	_, elem := retrieveStorageElement(record, s)

	if !strings.EqualFold(structs.GetKey(elem), "") { // check if the value was correctly retrieved
		stream.Send(&pb_storage.StorageRecordMessage{
			Nvalues: int32(len(structs.GetValues(elem))),
		})

		// we possibily have to send multiple values, so we loop over all of them
		//fmt.Printf("Expecting to send %v values\n", len(structs.GetValues(elem)))

		for _, value := range structs.GetValues(elem) {
			getErr := stream.Send(&pb_storage.StorageRecordMessage{
				Value:  []byte(value),
				Length: int64(len(value)),
			})
			if getErr != nil {
				log.Fatalf("Error while sending values %v\n", getErr)
			}
		}
	} else {
		log.Println("Fatal: the key requested was not found")
		stream.Send(&pb_storage.StorageRecordMessage{
			Nvalues: 0,
		})
		return nil
	}
	fmt.Println("Get successfully completed")
	return nil
}

/* Put RPC call. If the key already exists, the value associated is overwritten.
 * RPC call
 *
 * @param ctx: Context
 * @param putRecord: the new key:value record to save on the edge node
 */
func (s *storageServer) Put(ctx context.Context, putRecord *pb_storage.StorageRecordMessage) (*pb_storage.OpStatus, error) {
	fmt.Println("Received an rpc for a Put operation")

	newRecord := structs.Record{}                  // create a new Record
	structs.SetKey(&newRecord, putRecord.GetKey()) // set the new Record key, even if it will be saved on Dynamo
	key_present := false
	var elem structs.Record
	s.lock.RLock()
	for _, elem = range s.savedValues {
		if structs.GetKey(&elem) == putRecord.GetKey() {
			key_present = true
			break
		}
	}
	s.lock.RUnlock()

	ipSender := putRecord.GetIpSenderOwner()
	portSender := putRecord.GetPortSenderOwner()
	first_put := false
	if putRecord.GetForward() && !key_present { // message from client, I'm the owner of the key
		structs.SetOwner(&newRecord, true, s.address)
		first_put = true
	} else if key_present { // message from a server
		owner, ownerElemAddr := structs.GetOwner(&elem)
		if owner {
			if !putRecord.GetForward() { // two masters for the same key
				// received an update from another master (rule to no forward)
				// for a key of which the server is the owner
				// need of a reconciliation algorithm
				fmt.Println("Trying to resolve conflict")
				if int(putRecord.GetPortSenderOwner()) < s.address.Port ||
					(int(putRecord.GetPortSenderOwner()) == s.address.Port && strings.Compare(putRecord.GetIpSenderOwner(), s.address.Ip_addr) == -1) { // owner is the other server
					structs.SetOwner(&newRecord, false, structs.Socket{
						Ip_addr: putRecord.GetIpSenderOwner(),
						Port:    int(putRecord.GetPortSenderOwner()),
					})
					// fmt.Println("I am no longer the owner for the key: ", putRecord.GetKey())
				} else {
					structs.SetOwner(&newRecord, true, s.address)
				}
			} else {
				structs.SetOwner(&newRecord, true, s.address)
			}
		} else {
			if int(putRecord.GetPortSenderOwner()) != 0 && (int(putRecord.GetPortSenderOwner()) < s.address.Port ||
				(int(putRecord.GetPortSenderOwner()) == s.address.Port && strings.Compare(putRecord.GetIpSenderOwner(), s.address.Ip_addr) == -1)) { // owner is the other server
				structs.SetOwner(&newRecord, false, structs.Socket{
					Ip_addr: putRecord.GetIpSenderOwner(),
					Port:    int(putRecord.GetPortSenderOwner()),
				})
			} else {
				structs.SetOwner(&newRecord, false, ownerElemAddr)
			}
		}
	} else { // the key is not present and it's sent by the owner
		ownerAddr := structs.Socket{
			Ip_addr: putRecord.GetIpSenderOwner(),
			Port:    int(putRecord.GetPortSenderOwner()),
		}
		structs.SetOwner(&newRecord, false, ownerAddr)
	}

	// check if the size of the new object is compliant, otherwise directly save it on dynamo,
	// only if the server is the owner of the key
	owner, _ := structs.GetOwner(&elem)

	if putRecord.GetLength() > s.THRESHOLD_MAX_FILE {
		if owner {
			var tempArray = []string{}
			tempArray = append(tempArray, string(putRecord.GetValue()))
			server_utils.DynamoDBPutObject(&structs.DynamoObject{
				Key:  putRecord.Key,
				Vals: tempArray,
				Type: 1,
			})
		}
	} else { // the value can be stored locally by the server
		structs.AppendValue(&newRecord, string(putRecord.GetValue()))
		structs.SetTtl(&newRecord, initialTTL) // setting the ttl of the element
		structs.SetTimeSinceLastUpdate(&newRecord, 0)
		structs.IncrementTotalSize(&newRecord, int32(putRecord.GetLength()))

		if owner {
			dynamoRet := server_utils.DynamoDBDeleteObjects(putRecord.GetKey()) // check if the record is on the cloud and if so delete it
			if dynamoRet == -1 {
				log.Printf("Bad error while connecting to the cloud \n")
				return &pb_storage.OpStatus{Status: -1}, nil // simply returns nil, the operation is completed
			}
		}
	}
	var indexElem = -1
	for index, elem := range s.savedValues {
		if structs.GetKey(&elem) == putRecord.GetKey() { // change the value associated to the element
			indexElem = index
		}
	}

	if indexElem != -1 {
		s.lock.Lock() // lock before the insertion
		s.savedValues[indexElem] = newRecord
		s.lock.Unlock() // unlock after the insertion
	} else {
		s.lock.Lock() // lock before the insertion
		s.savedValues = append(s.savedValues, newRecord)
		s.lock.Unlock() // unlock after the insertion
	}
	if putRecord.GetForward() && first_put { // check if the server needs to contact its peers to send them the update
		connectToPeer(&pb_storage.StorageRecordMessage{
			Key:             structs.GetKey(&newRecord),
			Value:           putRecord.GetValue(),
			Opcode:          5,
			IpSenderOwner:   s.address.Ip_addr,
			PortSenderOwner: int32(s.address.Port),
			Forward:         putRecord.GetForward(),
		}, newRecord, s, ipSender, portSender)
	} else if putRecord.GetForward() && !first_put { // check if the server needs to contact its peers to send them the update
		connectToPeer(&pb_storage.StorageRecordMessage{
			Key:             structs.GetKey(&newRecord),
			Value:           putRecord.GetValue(),
			Opcode:          1,
			IpSenderOwner:   s.address.Ip_addr,
			PortSenderOwner: int32(s.address.Port),
			Forward:         putRecord.GetForward(),
		}, newRecord, s, ipSender, portSender)
	}

	fmt.Println("Put successfully completed")
	return &pb_storage.OpStatus{Status: 0}, nil // simply returns nil, the operation is completed
}

/* Del RPC call. Delete the key, if it does not exists returns an error message
 *
 * @param ctx: Context
 * @param delRes: StorageRecord instance, containing the key to delete
 */
func (s *storageServer) Del(ctx context.Context, delRecord *pb_storage.StorageRecordMessage) (*pb_storage.OpStatus, error) {
	fmt.Println("Received an rpc for a Del operation")
	var unlocked = false
	s.lock.RLock()
	ipSender := delRecord.GetIpSenderOwner()
	portSender := delRecord.GetPortSenderOwner()

	for index, record := range s.savedValues { // try to find a key corresponding to the client's given one
		if structs.GetKey(&record) == delRecord.GetKey() { // delete the record from the server struct
			owner, _ := structs.GetOwner(&record)
			unlocked = true
			s.lock.RUnlock()
			if len(structs.GetValues(&record)) > 0 {
				s.lock.Lock() // write lock before deletetion
				s.savedValues = append(s.savedValues[:index], s.savedValues[index+1:]...)
				s.lock.Unlock() // release lock
			} else {
				if owner {
					server_utils.DynamoDBDeleteObjects(delRecord.GetKey()) // Do this only if you have the key locally
				}
			}

			msg := pb_storage.OpStatus{
				Status:  0,
				Message: "Key:value successfully deleted",
			}
			if delRecord.GetForward() {
				connectToPeer(&pb_storage.StorageRecordMessage{
					Key:             delRecord.GetKey(),
					Value:           delRecord.GetValue(),
					Opcode:          3,
					IpSenderOwner:   s.address.Ip_addr,
					PortSenderOwner: int32(s.address.Port),
					Forward:         delRecord.GetForward(),
				}, record, s, ipSender, portSender)
			}
			return &msg, nil
		}
	}
	if !unlocked {
		s.lock.RUnlock()
	}
	fmt.Println("Del operation successfully completed")
	return &pb_storage.OpStatus{Status: -2}, nil
}

/* Append RPC call. Add the value to an existing key, if the key does not exists returns an error
 *
 * @param ctx: Context
 * @param appendRecord: StorageRecord instance, containing the key:value to append
 */
func (s *storageServer) Append(ctx context.Context, appendRecord *pb_storage.StorageRecordMessage) (*pb_storage.OpStatus, error) {
	fmt.Println("Received an rpc for an Append operation")

	index, elem := retrieveStorageElement(appendRecord, s)
	if structs.GetKey(elem) != "" {
		ipSender := appendRecord.GetIpSenderOwner()
		portSender := appendRecord.GetPortSenderOwner()

		structs.SetTtl(elem, initialTTL)
		structs.SetTimeSinceLastUpdate(elem, 0)
		owner, _ := structs.GetOwner(elem)
		if index != -1 {
			structs.AppendValue(elem, string(appendRecord.GetValue()))

			var length = int32(appendRecord.Length)

			// check if the Append makes the size becoming too huge. In that case
			// push the value on DynamoDB
			if structs.IncrementTotalSize(elem, int32(length)) > int32(s.THRESHOLD_MAX_FILE) && owner {
				server_utils.DynamoDBPutObject(&structs.DynamoObject{
					Key:  appendRecord.Key,
					Vals: structs.GetValues(elem),
				})
				structs.RemoveAll(elem) // simply clear the list
			}
			s.lock.Lock()
			s.savedValues[index] = *elem
			s.lock.Unlock()
		} else {
			if owner {
				res := server_utils.DynamoDBAppendValue(appendRecord.GetKey(), string(appendRecord.GetValue()))
				if res == 0 { // the Append sucessufully updated an Item on DynamoDB
					return &pb_storage.OpStatus{Status: 0}, nil
				} else {
					return &pb_storage.OpStatus{Status: -2}, nil // was -1
				}
			}
		}

		if appendRecord.GetForward() {
			connectToPeer(&pb_storage.StorageRecordMessage{
				Key:             appendRecord.Key,
				Value:           appendRecord.Value,
				Opcode:          4,
				IpSenderOwner:   s.address.Ip_addr,
				PortSenderOwner: int32(s.address.Port),
				Forward:         appendRecord.GetForward(),
			}, *elem, s, ipSender, portSender)
		}

		fmt.Println("Append successfully completed")
		return &pb_storage.OpStatus{
			Status: 0,
		}, nil // return a error record instead
	} else {
		fmt.Println("Error, the requested key does not exists")
		return &pb_storage.OpStatus{
			Status: -2,
		}, nil
	}
}

/* Update RPC call. Retrieve the values for the keys of which the server is the owner, returns nil if the key does not exists.
 * Server-side streaming RCP call
 *
 * @param ctx: Context
 * @param stream: a stream to get all the Update messages from the servers
 */
func (s *storageServer) Update(serverMsg *pb_storage.StorageRecordMessage, stream pb_storage.Storage_UpdateServer) error {
	fmt.Println("Received an rpc for a Update operation")

	// add the peer in the list of peers
	peerAddress.IpAddr = append(peerAddress.IpAddr, serverMsg.IpSenderOwner)
	peerAddress.Port = append(peerAddress.Port, serverMsg.PortSenderOwner)

	s.lock.RLock() // get the Read lock before starting to read the storage elements
	for _, elem := range s.savedValues {
		owner, ownerAddr := structs.GetOwner(&elem)
		if owner {
			stream.Send(&pb_storage.StorageRecordMessage{
				Key:             structs.GetKey(&elem),
				Nvalues:         int32(len(structs.GetValues(&elem))),
				Length:          1,
				IpSenderOwner:   ownerAddr.Ip_addr,
				PortSenderOwner: int32(ownerAddr.Port),
			})

			for _, value := range structs.GetValues(&elem) {
				getErr := stream.Send(&pb_storage.StorageRecordMessage{
					Value:  []byte(value),
					Length: int64(len(value)),
				})
				if getErr != nil {
					log.Fatalf("Error while sending values %v\n", getErr)
				}
			}
		}
	}
	stream.Send(&pb_storage.StorageRecordMessage{
		Length: 0,
	})
	s.lock.RUnlock() // release the read lock before exiting the RPC
	fmt.Println("Update successfully completed")
	return nil
}

/* Connect to the other servers to propagate operation received by a client, in order to increment scalability with respect
 * to the data and the number of users
 *
 * @param record: StorageRecord instance, containing the operation to perform
 * @param savedRecord: the Record, containing information to send to the peers
 * @param s: storageServer struct instance
 * @param ipSender: the ip address of the server sending the record
 * @param portSender: the port of the server sending the record
 */
func connectToPeer(storageRecord *pb_storage.StorageRecordMessage, savedRecord structs.Record, s *storageServer,
	ipSender string, portSender int32) int {

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())

	// connection with peer
	owner, ownerAddr := structs.GetOwner(&savedRecord)
	if owner && storageRecord.Forward { //|| storageRecord.Opcode == 5 { // send all the message record, except the one that send it to the master
		var ret int
		storageRecord.Forward = false
		for index := 0; index < len(peerAddress.IpAddr); index++ {
			if (strings.Compare(peerAddress.IpAddr[index], storageRecord.IpSenderOwner) != 0 || peerAddress.Port[index] != storageRecord.PortSenderOwner) &&
				(strings.Compare(peerAddress.IpAddr[index], ipSender) != 0 || peerAddress.Port[index] != portSender || storageRecord.Opcode == 5) {
				peerIpAddr := peerAddress.IpAddr[index]
				peerPort := peerAddress.Port[index]

				serverConf := peerIpAddr + ":" + fmt.Sprint(peerPort)
				// fmt.Println("Trying to connect with peer: ", serverConf)
				conn, err := grpc.Dial(serverConf, opts...) // establish a channel between client and server
				if err != nil {
					log.Fatalf("Error while estabishing connection with the peer on port: %v", peerPort)
				}
				defer conn.Close()
				peerClient := pb_storage.NewStorageClient(conn)

				_, ret = client_utils.AtLestOnceRetx(int(storageRecord.GetOpcode()), &peerClient, storageRecord, true)
			}
		}
		if ret == 0 {
			return ret
		}
	} else if !owner {
		storageRecord.Forward = true

		ownerIpAddr := ownerAddr.Ip_addr
		ownerPort := ownerAddr.Port

		ownerConf := ownerIpAddr + ":" + fmt.Sprint(ownerPort)
		// fmt.Println("Trying to connect with peer: ", ownerConf)
		conn, err := grpc.Dial(ownerConf, opts...) // establish a channel between client and server
		if err != nil {
			log.Fatalf("Error while estabishing connection with the peer on port: %v", ownerPort)
		}
		defer conn.Close()
		peerClient := pb_storage.NewStorageClient(conn)
		_, ret := client_utils.AtLestOnceRetx(int(storageRecord.GetOpcode()), &peerClient, storageRecord, true)
		if ret == 0 {
			return ret
		}
	}
	return 0
}

/*--------------------------------------------- Register functions ----------------------------------------------------------*/

/* Connect to the register node to register itself and get the list of available peers
 *
 * @param connection: RegisterClient instance
 * @param Info: StorageServerMsg struct
 */
func connectToRegister(connection pb_register.RegisterClient, Info *pb_register.StorageServerMsg) (*pb_register.ListOfStorageServerMsg, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	peer, err := connection.GetListOfPeers(ctx, Info)
	if err != nil {
		log.Fatalf("Error during GetListOfPeers(): %v", err)
	}
	return peer, nil
}

/* Send the hearthbeat to the register, to notify that the edge node is still alive
 *
 * @param conncetion: RegisterClient instance
 * @param Info: StorageServerMsg instance
 */
func sendHeartbeat(connection pb_register.RegisterClient, Info *pb_register.StorageServerMsg) (*pb_register.ListOfStorageServerMsg, error) {
	for {
		time.Sleep(60000000000.0)
		fmt.Print("Sending an heartbeat\n")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		var err error
		peerAddress, err = connection.Heartbeat(ctx, Info)
		if err != nil {
			if strings.Contains(err.Error(), "Unavailable desc") {
				fmt.Printf("Error during send Heartbeat...\n")
			} else {
				log.Fatalf("Error during send Heartbeat: %v", err)
			}
		}
		cancel()
	}
}

/*--------------------------------------------- Utility functions -----------------------------------------------------------*/

/* Retrieves a key:[value] doing as the following
 * - first, it checks if the requested keys is present in the system
 * - if it is and the value is > 0, this means that the server has the values locally stored
 * - if the value is 0, it goes on Dynamo to retrieve the values of the key stored on cloud
 * If it has not the key, then an empty is returned

 * @param record: the StorageRecord containing information about the Record that needs to be retrieved
 * @param s: stroageServer struct instance
 */
func retrieveStorageElement(record *pb_storage.StorageRecordMessage, s *storageServer) (int, *structs.Record) {
	s.lock.RLock()
	for index, elem := range s.savedValues {

		// Key found, if the value list is empty then it is saved on Dynamo and so it needs to be retrieved
		if structs.GetKey(&elem) == record.GetKey() {
			if len(structs.GetValues(&elem)) > 0 {
				structs.SetTtl(&elem, initialTTL) // to update the ttl of the element
				// fmt.Println("The record was found locally")
				s.lock.RUnlock()
				return index, &elem
			} else {
				// fmt.Println("Going to search requested key on Dynamo...")
				// retrieve the record from Dynamo
				retrRecord := server_utils.DynamoDBRetrieveObjects(&structs.DynamoObject{
					Key: record.Key,
				})
				owner, ownerAddr := structs.GetOwner(&s.savedValues[index])
				structs.SetOwner(retrRecord, owner, ownerAddr)
				// update the TotalSize of the values
				for _, value := range structs.GetValues(retrRecord) {
					structs.IncrementTotalSize(retrRecord, int32(len(value)))
				}
				if structs.GetTotalSize(retrRecord) < int32(s.THRESHOLD_MAX_FILE) {

					structs.SetTtl(retrRecord, initialTTL)
					// I have to delete it before append it
					s.savedValues = append(s.savedValues[:index], s.savedValues[index+1:]...)
					s.savedValues = append(s.savedValues, *retrRecord)

					s.lock.RUnlock()
					return len(s.savedValues) - 1, retrRecord
				}
				s.lock.RUnlock()
				return -1, retrRecord
			}
		}
	}

	// Simply return an empty record, that tells that the key is not present on this edge node
	var empyRecord = structs.Record{}
	s.lock.RUnlock()
	structs.SetKey(&empyRecord, "")
	return -1, &empyRecord
}

/* Return the IP address of the current edge node */
func getMyIp() string {
	var ret string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalf("Oops: " + err.Error() + "\n")
		os.Exit(1)
	}

	for _, ip := range addrs {
		if ipnet, ok := ip.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if len(ip.String()) <= 19 {
				ipAddr := ip.String()
				ret = strings.Split(ipAddr, "/")[0]
			}
		}
	}
	return ret
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

/* Decrement TTLs of the Record periodically
 *
 *@param s: server instance, used to access the currently saved key:[value]
 */
func decrementRecordsTTL(s *storageServer) {
	for {
		time.Sleep(60000000000.0)
		for k := range s.savedValues {
			structs.DecrementTTL(&s.savedValues[k], 60.0)
			if structs.GetTimeSinceLastUpdate(&s.savedValues[k]) < maxTimeSinceLastUpdate {
				structs.IncreaseTimeSinceLastUpdate(&s.savedValues[k], 60.0)
				owner, _ := structs.GetOwner(&s.savedValues[k])
				if owner && structs.GetTimeSinceLastUpdate(&s.savedValues[k]) >= maxTimeSinceLastUpdate {
					// fmt.Printf("key %v loaded on Dynamo\n", structs.GetKey(&s.savedValues[k]))
					server_utils.DynamoDBPutObject(&structs.DynamoObject{
						Key:  structs.GetKey(&s.savedValues[k]),
						Vals: structs.GetValues(&s.savedValues[k]),
					})
				}
			}
			if structs.GetTtl(&s.savedValues[k]) <= 0.0 && structs.GetTotalSize(&s.savedValues[k]) > 0 {
				structs.RemoveAll(&s.savedValues[k])
			}
		}
	}
}

/* Receive an update message form one of the connected peers
 *
 * @param s: storageServer struct
 * @param myIpAddr: IP address of the current edge node
 * @param myPort: port number of the current process
 * @param ipAddr: peer IP address
 * @param port: peer port number
 */
func getUpdate(s *storageServer, myIpAddr string, myPort int, ipAddr string, port int) int {
	fmt.Println("Serving an Update...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())

	connection, err := grpc.Dial(ipAddr+":"+strconv.Itoa(port), opts...) // establish a channel between StorageServer and Register
	if err != nil {
		return -1
	}
	client := pb_storage.NewStorageClient(connection)
	stream, err := client.Update(ctx, &pb_storage.StorageRecordMessage{
		IpSenderOwner:   myIpAddr,
		PortSenderOwner: int32(myPort),
	})
	if err != nil {
		return -1
	}

	for {
		// Set up for receive the values
		getRecord, err := stream.Recv() // this is the record containing the number of values
		if err != nil {
			return -1
		}
		if getRecord.Length == 0 {
			break
		}
		newSerRecord := &structs.Record{}
		key := getRecord.Key
		recordIpAddr := getRecord.IpSenderOwner
		recordPort := getRecord.PortSenderOwner
		structs.SetKey(newSerRecord, key)
		structs.SetOwner(newSerRecord, false, structs.Socket{
			Ip_addr: recordIpAddr,
			Port:    int(recordPort),
		})
		// fmt.Println("Receiving values associated to key:", key)

		// Iterate until each value is received
		for index := 0; index < int(getRecord.Nvalues); index++ {
			getRecord, err := stream.Recv() // receive the value from the RPC
			if err != nil {
				return -1
			}
			structs.AppendValue(newSerRecord, string(getRecord.Value))
		}
		s.savedValues = append(s.savedValues, *newSerRecord)
	}
	fmt.Println("Update successfully completed")
	return 0
}

// Constructor for the rpc server
func newServer(folder string, ipAddr string, port int) *storageServer {
	s := &storageServer{}
	s.folder = folder
	s.address = structs.Socket{
		Ip_addr: ipAddr,
		Port:    port,
	}
	s.THRESHOLD_MAX_FILE = 20000 // the max length that a single value can have
	s.lock = sync.RWMutex{}      //initialize the read-write lock, to handle concurrent operations
	return s
}

/*--------------------------------------------- Main function --------------------------------------------------------------*/

func main() {
	registerPort := "1234"
	if len(os.Args) < 2 {
		log.Fatalf("Please specify the register IP address")
	}
	registerAddr := os.Args[1]

	var opts1 []grpc.DialOption
	opts1 = append(opts1, grpc.WithInsecure())
	opts1 = append(opts1, grpc.WithBlock())

	conn, err := grpc.Dial(registerAddr+":"+registerPort, opts1...) // establish a channel between StorageServer and Register

	if err != nil {
		log.Fatalf("Error in the connection with RegisterServer: %v", err)
	}
	client := pb_register.NewRegisterClient(conn)
	port, err := GetFreePort()
	if err != nil {
		log.Fatalf("Error during GetFreePort(): %v", err)
	}
	ipAddr := getMyIp()

	Info = pb_register.StorageServerMsg{
		IpAddr: ipAddr,
		Port:   int32(port),
	}

	peerAddress, err = connectToRegister(client, &Info)
	if err != nil {
		log.Fatalf("Error during get information about peers: %v", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(ipAddr+":%d", port))
	if err != nil {
		log.Fatalf("Error during listening: %v", err)
	}

	// create the new directory (tmp folder, that will contain little files for now)
	var dirPath string = "server_" + fmt.Sprint(port)
	err = os.Mkdir(dirPath, 0755)
	if err != nil {
		log.Fatalf("Failed to create server dir, %v", err.Error())
	}

	// start the server to receive upcoming requests from the clients
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	s := newServer(dirPath, ipAddr, port)

	for index := 0; index < len(peerAddress.IpAddr); index++ {
		getUpdate(s, ipAddr, port, peerAddress.IpAddr[index], int(peerAddress.Port[index]))
	}

	pb_storage.RegisterStorageServer(grpcServer, s)
	fmt.Println("The server is listening on port:", port)

	go sendHeartbeat(client, &Info)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGINT)
	go func() {
		<-c
		err := os.RemoveAll(dirPath)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(1)
	}()

	go decrementRecordsTTL(s)

	grpcServer.Serve(lis)
}
