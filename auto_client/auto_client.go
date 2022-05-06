/* This file contains an automatic client that does RPC request based on a fixed workload. This type of client is usefull to
 * test the performance of the distributed system, and has the following fixed workload operation modes:
 *	- 85% Get operations, 15% Put operations
 *	- 40% Put operations, 20% Append operations, 40% Get operations
 *
 * In both cases, we consider different keys and values size, so that we can also test the interaction with the cloud
 */

package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/juju/fslock"

	utils "github.com/piercirocaliandro/sdcc-edge-computing/client_utils"
	pb_storage "github.com/piercirocaliandro/sdcc-edge-computing/pb_storage"
)

const ops = 50 // a hundred operation for each node
const clients = 30
const servers = 15

var port string = "1234"

func debug(addr string, key string, value string, firstPut int) {
	conn := utils.SetUpConnection(addr, port)
	testClient := pb_storage.NewStorageClient(conn)
	utils.AtLestOnceRetx(1, &testClient, &pb_storage.StorageRecordMessage{
		Key:     key,
		Value:   []byte(value),
		Forward: true,
	}, true)
	conn.Close()
}

/* Implement the first type of workload */
func Workload1(addr string, key string, value string, firstPut int) {
	var randPut = 8 // the number of Put operation to perform
	var puts = []int{}
	puts = append(puts, firstPut) // set the first Put operation

	if firstPut == 0 {
		// Generate and set a random Exponential distributed sleep time
		var duration = int(-0.5*math.Log(1.0-rand.Float64())*math.Pow10(6)) * int(time.Microsecond)
		time.Sleep(time.Duration(duration))
		conn := utils.SetUpConnection(addr, port)
		testClient := pb_storage.NewStorageClient(conn)
		utils.AtLestOnceRetx(1, &testClient, &pb_storage.StorageRecordMessage{
			Key:     key,
			Value:   []byte(value),
			Forward: true,
		}, true)
		conn.Close()
		time.Sleep(3 * time.Second)
	} else {
		time.Sleep(5 * time.Second)
	}

	conn := utils.SetUpConnection(addr, port)
	defer conn.Close()
	testClient := pb_storage.NewStorageClient(conn)

	rand.Seed(int64(os.Getpid()))

	// Generate the random Put operations
	var i = 0
	for i < randPut-1 {
		randNum := rand.Intn(50)
		for index, elem := range puts {
			if elem == randNum || randNum <= firstPut {
				break
			}
			if index == len(puts)-1 {
				puts = append(puts, randNum)
				i += 1
			}
		}
	}

	// Local variable usefull to compute the output data
	var startTime time.Time
	var startTimeGlobal time.Time
	var endTime time.Duration
	var requestTime time.Duration
	var data = ""
	var isPut = false // tells if the current operation was a Put or a Get

	/* Execute the actual payload. Simply, if the index is odd, send an "heavier" value */
	startTimeGlobal = time.Now()
	for i := 0; i < ops; i++ {
		// Generate and set a random Exponential distributed sleep time
		var duration = int(-2.5*math.Log(1.0-rand.Float64())*math.Pow10(6)) * int(time.Microsecond)
		time.Sleep(time.Duration(duration))
		for _, elem := range puts {
			if elem == i { // Perform a Put operation
				isPut = true
				requestTime = time.Since(startTimeGlobal)
				startTime = time.Now()
				//secs := startTime.Unix()
				utils.AtLestOnceRetx(1, &testClient, &pb_storage.StorageRecordMessage{
					Key:     key,
					Value:   []byte(value),
					Forward: true,
				}, true)
				endTime = time.Since(startTime)
				data = data + "Put," + "1," + key + "," + requestTime.String() + "," +
					endTime.String() + "," + fmt.Sprint(clients) + "," + fmt.Sprint(servers) + "," +
					fmt.Sprint(ops) + ",2.5" + "\n"

			}
		}
		if !isPut {
			startTime = time.Now()
			requestTime = time.Since(startTimeGlobal)
			utils.AtLestOnceRetx(2, &testClient, &pb_storage.StorageRecordMessage{
				Key:     key,
				Forward: true,
			}, true)
			endTime = time.Since(startTime)
			data = data + "Get," + "1," + key + "," + requestTime.String() + "," +
				endTime.String() + "," + fmt.Sprint(clients) + "," + fmt.Sprint(servers) + "," +
				fmt.Sprint(ops) + ",2.5" + "\n"
		}
		isPut = false
	}
	writeDataOnFile(data)
}

/* Implement the second type of workload */
func Workload2(addr string, key string, value string, firstPut int) {
	var randPut = []int{}
	var randAppend = []int{}
	randPut = append(randPut, firstPut) // set the first Put operation
	randAppend = append(randAppend, 5)

	var puts = 20
	var appends = 10

	rand.Seed(int64(os.Getpid()))

	var i = 0
	for i < puts-1 {
		randNum := rand.Intn(50)
		if randNum != 5 && randNum > firstPut {
			for index, elem := range randPut {
				if elem == randNum {
					break
				}
				if index == len(randPut)-1 {
					randPut = append(randPut, randNum)
					i += 1
				}
			}
		}
	}

	i = 0
	for i < appends-1 {
		randNum := rand.Intn(50)
		for index, elem := range randPut {
			if elem == randNum {
				break
			}
			for index2, elem2 := range randAppend {
				if elem2 == randNum {
					break
				}
				if index == len(randPut)-1 && index2 == len(randAppend)-1 {
					randAppend = append(randAppend, randNum)
					i += 1
				}
			}
		}
	}

	var startTime time.Time
	var startTimeGlobal time.Time
	var endTime time.Duration
	var requestTime time.Duration
	var data = ""
	var isWrite = false // tells if the current operation was a Put or a Get

	conn := utils.SetUpConnection(addr, port)
	defer conn.Close()
	testClient := pb_storage.NewStorageClient(conn)

	if firstPut == 0 {
		// Generate and set a random Exponential distributed sleep time
		var duration = int(-0.5*math.Log(1.0-rand.Float64())*math.Pow10(6)) * int(time.Microsecond)
		time.Sleep(time.Duration(duration))
		conn := utils.SetUpConnection(addr, port)
		testClient := pb_storage.NewStorageClient(conn)
		utils.AtLestOnceRetx(1, &testClient, &pb_storage.StorageRecordMessage{
			Key:     key,
			Value:   []byte(value),
			Forward: true,
		}, true)
		conn.Close()
		time.Sleep(3 * time.Second)
	} else {
		time.Sleep(5 * time.Second)
	}

	startTimeGlobal = time.Now()
	for i := 0; i < ops; i++ {
		// Generate and set a random Exponential distributed sleep time
		var duration = int(-2.5*math.Log(1.0-rand.Float64())*math.Pow10(6)) * int(time.Microsecond)
		time.Sleep(time.Duration(duration))
		for _, elem := range randPut {
			if elem == i { // Perform a Put operation
				isWrite = true
				startTime = time.Now()
				requestTime = time.Since(startTimeGlobal)
				utils.AtLestOnceRetx(1, &testClient, &pb_storage.StorageRecordMessage{
					Key:     key,
					Value:   []byte(value),
					Forward: true,
				}, true)
				endTime = time.Since(startTime)
				data = data + "Put," + "2," + key + "," + requestTime.String() + "," +
					endTime.String() + "," + fmt.Sprint(clients) + "," + fmt.Sprint(servers) + "," +
					fmt.Sprint(ops) + ",2.5" + "\n"
			}
		}
		for _, elem := range randAppend {
			if elem == i {
				isWrite = true
				startTime = time.Now()
				requestTime = time.Since(startTimeGlobal)
				utils.AtLestOnceRetx(4, &testClient, &pb_storage.StorageRecordMessage{
					Key:     key,
					Value:   []byte(value),
					Forward: true,
				}, true)
				endTime = time.Since(startTime)
				data = data + "Append," + "2," + key + "," + requestTime.String() + "," +
					endTime.String() + "," + fmt.Sprint(clients) + "," + fmt.Sprint(servers) + "," +
					fmt.Sprint(ops) + ",2.5" + "\n"
			}
		}
		if !isWrite {
			startTime = time.Now()
			requestTime = time.Since(startTimeGlobal)
			utils.AtLestOnceRetx(2, &testClient, &pb_storage.StorageRecordMessage{
				Key:     key,
				Forward: true,
			}, true)
			endTime = time.Since(startTime)
			data = data + "Get," + "2," + key + "," + requestTime.String() + "," +
				endTime.String() + "," + fmt.Sprint(clients) + "," + fmt.Sprint(servers) + "," +
				fmt.Sprint(ops) + ",2.5" + "\n"
		}
		isWrite = false // reset the previous op type
	}

	writeDataOnFile(data)
}

func main() {
	if len(os.Args) < 6 {
		log.Fatalf("Usage: register IP, a key name, a value, a mode of operation and a first Put number")
	}
	opMode, _ := strconv.Atoi(os.Args[4]) // convert the operation mode
	firstPut, _ := strconv.Atoi(os.Args[5])

	switch opMode {
	case 1: // Workload 1, Put
		Workload1(os.Args[1], os.Args[2], os.Args[3], firstPut)
	case 2: // Workload 2, Get
		Workload2(os.Args[1], os.Args[2], os.Args[3], firstPut)
	case 3:
		debug(os.Args[1], os.Args[2], os.Args[3], firstPut)
	}
}

func writeDataOnFile(data string) {
	lock := fslock.New("perf_data.csv")
	lockErr := lock.TryLock()

	for lockErr != nil {
		fmt.Println("falied to acquire lock > " + lockErr.Error())
		lockErr = lock.TryLock()
	}
	/*if lockErr != nil {
		return
	}*/
	fmt.Println("got the lock")

	// Write data on file ...
	file, err := os.OpenFile("perf_data.csv", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error while opening file:", err)
	}
	file.Write([]byte(data))

	// release the lock
	lock.Unlock()
}
