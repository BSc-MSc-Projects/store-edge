/* A set of Get, Put, Append e Del operations carried out with the aim of
 * testing the correct functioning of the system in differents scenarios.
 * Focus is on demonstrating the consistency reached between all the nodes
 * following the period necessary to propagate the informations.
 */

package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	utils "github.com/piercirocaliandro/sdcc-edge-computing/client_utils"
	pb_storage "github.com/piercirocaliandro/sdcc-edge-computing/pb_storage"
	structs "github.com/piercirocaliandro/sdcc-edge-computing/util_structs"
)

var regAddr = "172.27.174.218"
var regPort = "1234"

/* Test to verify that the operations are correctly saved and retrieved locally for a single client*/
func TestLocalSaveSingleClient(t *testing.T) {
	var size = 25

	assert := assert.New(t)
	conn := utils.SetUpConnection(regAddr, regPort)
	defer conn.Close()
	client := pb_storage.NewStorageClient(conn)

	value := make([]byte, 0)

	for i := 0; i < size; i++ {
		value = append(value, byte('a'))
	}
	localSavedRecord := structs.Record{}
	structs.SetKey(&localSavedRecord, "key")
	structs.AppendValue(&localSavedRecord, string(value))

	// Put operation
	fmt.Println("Going with the put")
	_, res := utils.AtLestOnceRetx(1, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)

	// Get operation
	record, res := utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the first Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the first Append operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)
	structs.AppendValue(&localSavedRecord, string(value))

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the second Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the 3rd Append operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)
	structs.AppendValue(&localSavedRecord, string(value))

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the 2nd Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the last Append operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)
	structs.AppendValue(&localSavedRecord, string(value))

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the last Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	localSavedRecord2 := structs.Record{}
	structs.SetKey(&localSavedRecord2, "key")
	structs.AppendValue(&localSavedRecord2, string(value))

	// Put operation
	fmt.Println("Going with the put")
	_, res = utils.AtLestOnceRetx(1, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the first Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord2), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord2), structs.GetValues(record), "The two values should be the same.")

	// Finally, delete operation
	_, res = utils.AtLestOnceRetx(3, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the final Del operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)
}

/* Test to verify that the operations are correctly done on DynamoDB for a single client*/
func TestDynamoIntegrationSingleClient(t *testing.T) {
	var size = 25000

	assert := assert.New(t)
	conn := utils.SetUpConnection(regAddr, regPort)
	defer conn.Close()
	client := pb_storage.NewStorageClient(conn)

	value := make([]byte, 0)

	for i := 0; i < size; i++ {
		value = append(value, byte('a'))
	}
	localSavedRecord := structs.Record{}
	structs.SetKey(&localSavedRecord, "key")
	structs.AppendValue(&localSavedRecord, string(value))

	// Put operation
	fmt.Println("Going with the put")
	_, res := utils.AtLestOnceRetx(1, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the operation")
		t.Fail()
	}
	time.Sleep(3 * time.Second)

	// Get operation
	record, res := utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the first Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the first Append operation")
		t.Fail()
	}
	time.Sleep(3 * time.Second)
	structs.AppendValue(&localSavedRecord, string(value))

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the second Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the 3rd Append operation")
		t.Fail()
	}
	time.Sleep(3 * time.Second)
	structs.AppendValue(&localSavedRecord, string(value))

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the 2nd Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the last Append operation")
		t.Fail()
	}
	time.Sleep(3 * time.Second)
	structs.AppendValue(&localSavedRecord, string(value))

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the last Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	localSavedRecord2 := structs.Record{}
	structs.SetKey(&localSavedRecord2, "key")
	structs.AppendValue(&localSavedRecord2, string(value))

	// Put operation
	fmt.Println("Going with the put")
	_, res = utils.AtLestOnceRetx(1, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the operation")
		t.Fail()
	}
	time.Sleep(3 * time.Second)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the first Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord2), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord2), structs.GetValues(record), "The two values should be the same.")

	// Finally, delete operation
	_, res = utils.AtLestOnceRetx(3, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the final Del operation")
		t.Fail()
	}
	time.Sleep(3 * time.Second)
}

/* Test to verify that the operations are correctly saved and retrieved locally for multiple clients and servers*/
func TestLocalSaveMultipleClients(t *testing.T) {
	var size = 25

	assert := assert.New(t)
	conn := utils.SetUpConnection(regAddr, regPort)
	defer conn.Close()
	client := pb_storage.NewStorageClient(conn)

	value := make([]byte, 0)

	for i := 0; i < size; i++ {
		value = append(value, byte('a'))
	}
	localSavedRecord := structs.Record{}
	structs.SetKey(&localSavedRecord, "key")
	structs.AppendValue(&localSavedRecord, string(value))

	// Put operation
	fmt.Println("Going with the put")
	_, res := utils.AtLestOnceRetx(1, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)
	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res := utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the first Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the first Append operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)
	structs.AppendValue(&localSavedRecord, string(value))

	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the second Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the 3rd Append operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)
	structs.AppendValue(&localSavedRecord, string(value))

	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the 2nd Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the last Append operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)
	structs.AppendValue(&localSavedRecord, string(value))

	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the last Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	localSavedRecord2 := structs.Record{}
	structs.SetKey(&localSavedRecord2, "key")
	structs.AppendValue(&localSavedRecord2, string(value))

	// Put operation
	fmt.Println("Going with the put")
	_, res = utils.AtLestOnceRetx(1, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)

	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the first Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord2), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord2), structs.GetValues(record), "The two values should be the same.")

	// Finally, delete operation
	_, res = utils.AtLestOnceRetx(3, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the final Del operation")
		t.Fail()
	}

	conn.Close()
}

/* Test to verify that the operations are correctly done on DynamoDB for multiple clients and servers*/
func TestDynamoIntegrationMultipleClients(t *testing.T) {
	var size = 25000

	assert := assert.New(t)
	conn := utils.SetUpConnection(regAddr, regPort)
	defer conn.Close()
	client := pb_storage.NewStorageClient(conn)

	value := make([]byte, 0)

	for i := 0; i < size; i++ {
		value = append(value, byte('a'))
	}
	localSavedRecord := structs.Record{}
	structs.SetKey(&localSavedRecord, "key")
	structs.AppendValue(&localSavedRecord, string(value))

	// Put operation
	fmt.Println("Going with the put")
	_, res := utils.AtLestOnceRetx(1, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the operation")
		t.Fail()
	}
	conn.Close()
	time.Sleep(3 * time.Second)

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res := utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the first Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the first Append operation")
		t.Fail()
	}
	time.Sleep(3 * time.Second)
	structs.AppendValue(&localSavedRecord, string(value))

	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the second Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the 3rd Append operation")
		t.Fail()
	}
	time.Sleep(3 * time.Second)
	structs.AppendValue(&localSavedRecord, string(value))

	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the 2nd Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the last Append operation")
		t.Fail()
	}
	time.Sleep(3 * time.Second)
	structs.AppendValue(&localSavedRecord, string(value))

	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the last Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	localSavedRecord2 := structs.Record{}
	structs.SetKey(&localSavedRecord2, "key")
	structs.AppendValue(&localSavedRecord2, string(value))

	// Put operation
	fmt.Println("Going with the put")
	_, res = utils.AtLestOnceRetx(1, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the operation")
		t.Fail()
	}
	time.Sleep(3 * time.Second)
	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the first Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord2), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord2), structs.GetValues(record), "The two values should be the same.")

	// Finally, delete operation
	_, res = utils.AtLestOnceRetx(3, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the final Del operation")
		t.Fail()
	}
	conn.Close()
}

/* Test to verify that the operations are correctly saved and retrieved locally for multiple clients and servers*/
func TestLocalSaveMultipleClientsMultipleKeys(t *testing.T) {
	var size = 25

	assert := assert.New(t)
	conn := utils.SetUpConnection(regAddr, regPort)
	defer conn.Close()
	client := pb_storage.NewStorageClient(conn)

	value := make([]byte, 0)

	for i := 0; i < size; i++ {
		value = append(value, byte('a'))
	}
	localSavedRecord := structs.Record{}
	structs.SetKey(&localSavedRecord, "key")
	structs.AppendValue(&localSavedRecord, string(value))

	// Put operation
	fmt.Println("Going with the put")
	_, res := utils.AtLestOnceRetx(1, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)
	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res := utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the first Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	// Put operation
	fmt.Println("Going with the put")
	_, res = utils.AtLestOnceRetx(1, &client, &pb_storage.StorageRecordMessage{
		Key:     "key2",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key2",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the first Append operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)
	localSavedRecord2 := structs.Record{}
	structs.SetKey(&localSavedRecord2, "key2")
	structs.AppendValue(&localSavedRecord2, string(value))
	structs.AppendValue(&localSavedRecord2, string(value))

	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key2"}, true)
	if res != 0 {
		t.Log("Error in the second Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord2), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord2), structs.GetValues(record), "The two values should be the same.")

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the 3rd Append operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)
	structs.AppendValue(&localSavedRecord, string(value))

	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the 2nd Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key2",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the last Append operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)
	structs.AppendValue(&localSavedRecord2, string(value))

	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key2"}, true)
	if res != 0 {
		t.Log("Error in the last Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord2), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord2), structs.GetValues(record), "The two values should be the same.")

	localSavedRecord3 := structs.Record{}
	structs.SetKey(&localSavedRecord3, "key")
	structs.AppendValue(&localSavedRecord3, string(value))

	// Put operation
	fmt.Println("Going with the put")
	_, res = utils.AtLestOnceRetx(1, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the operation")
		t.Fail()
	}
	time.Sleep(1 * time.Second)
	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the first Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord3), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord3), structs.GetValues(record), "The two values should be the same.")

	// Finally, delete operation
	_, res = utils.AtLestOnceRetx(3, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the final Del operation")
		t.Fail()
	}
	time.Sleep(2 * time.Second)

	// Finally, delete operation
	_, res = utils.AtLestOnceRetx(3, &client, &pb_storage.StorageRecordMessage{
		Key:     "key2",
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the final Del operation")
		t.Fail()
	}
	conn.Close()
}

/* Test to verify that the operations are correctly done on DynamoDB for multiple clients and servers*/
func TestDynamoIntegrationMultipleClientsMultipleKeys(t *testing.T) {
	var size = 25000

	assert := assert.New(t)
	conn := utils.SetUpConnection(regAddr, regPort)
	defer conn.Close()
	client := pb_storage.NewStorageClient(conn)

	value := make([]byte, 0)

	for i := 0; i < size; i++ {
		value = append(value, byte('a'))
	}
	localSavedRecord := structs.Record{}
	structs.SetKey(&localSavedRecord, "key")
	structs.AppendValue(&localSavedRecord, string(value))

	// Put operation
	fmt.Println("Going with the put")
	_, res := utils.AtLestOnceRetx(1, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the operation")
		t.Fail()
	}
	time.Sleep(3 * time.Second)
	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res := utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the first Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	// Put operation
	fmt.Println("Going with the put")
	_, res = utils.AtLestOnceRetx(1, &client, &pb_storage.StorageRecordMessage{
		Key:     "key2",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the operation")
		t.Fail()
	}
	time.Sleep(3 * time.Second)

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key2",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the first Append operation")
		t.Fail()
	}
	time.Sleep(3 * time.Second)
	localSavedRecord2 := structs.Record{}
	structs.SetKey(&localSavedRecord2, "key2")
	structs.AppendValue(&localSavedRecord2, string(value))
	structs.AppendValue(&localSavedRecord2, string(value))

	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key2"}, true)
	if res != 0 {
		t.Log("Error in the second Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord2), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord2), structs.GetValues(record), "The two values should be the same.")

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the 3rd Append operation")
		t.Fail()
	}
	time.Sleep(3 * time.Second)
	structs.AppendValue(&localSavedRecord, string(value))

	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the 2nd Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord), structs.GetValues(record), "The two values should be the same.")

	// Append operation
	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "key2",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the last Append operation")
		t.Fail()
	}
	time.Sleep(3 * time.Second)
	structs.AppendValue(&localSavedRecord2, string(value))

	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key2"}, true)
	if res != 0 {
		t.Log("Error in the last Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord2), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord2), structs.GetValues(record), "The two values should be the same.")

	localSavedRecord3 := structs.Record{}
	structs.SetKey(&localSavedRecord3, "key")
	structs.AppendValue(&localSavedRecord3, string(value))

	// Put operation
	fmt.Println("Going with the put")
	_, res = utils.AtLestOnceRetx(1, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Value:   []byte(value),
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the operation")
		t.Fail()
	}
	time.Sleep(3 * time.Second)
	conn.Close()

	conn = utils.SetUpConnection(regAddr, regPort)
	client = pb_storage.NewStorageClient(conn)

	// Get operation
	record, res = utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key: "key"}, true)
	if res != 0 {
		t.Log("Error in the first Get operation")
		t.Fail()
	}
	assert.Equal(structs.GetKey(&localSavedRecord3), structs.GetKey(record), "The two keys should be the same.")
	assert.Equal(structs.GetValues(&localSavedRecord3), structs.GetValues(record), "The two values should be the same.")

	// Finally, delete operation
	_, res = utils.AtLestOnceRetx(3, &client, &pb_storage.StorageRecordMessage{
		Key:     "key",
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the final Del operation")
		t.Fail()
	}

	// Finally, delete operation
	_, res = utils.AtLestOnceRetx(3, &client, &pb_storage.StorageRecordMessage{
		Key:     "key2",
		Forward: true}, true)
	if res != 0 {
		t.Log("Error in the final Del operation")
		t.Fail()
	}
	conn.Close()
}

/* Test that the RPC on an element that does not exists return a proper error code */
func TestFailGetDel(t *testing.T) {

	assert := assert.New(t)

	// Set up the connection
	conn := utils.SetUpConnection(regAddr, regPort)
	defer conn.Close()
	client := pb_storage.NewStorageClient(conn)

	_, res := utils.AtLestOnceRetx(2, &client, &pb_storage.StorageRecordMessage{
		Key:   "notex",
		Value: []byte("")}, true)
	if res == 0 {
		t.Log("Error on Get: expected a res != 0, got ", res)
		t.Fail()
	}
	assert.NotEqual(res, 0)

	_, res = utils.AtLestOnceRetx(4, &client, &pb_storage.StorageRecordMessage{
		Key:     "notex",
		Value:   []byte(""),
		Forward: true}, true)
	if res == 0 {
		t.Log("Error on Append: expected a res != 0, got ", res)
		t.Fail()
	}
	assert.NotEqual(res, 0)

	_, res = utils.AtLestOnceRetx(3, &client, &pb_storage.StorageRecordMessage{
		Key:     "notex",
		Value:   []byte(""),
		Forward: true}, true)
	if res == 0 {
		t.Log("Error on Del: expected a res != 0, got ", res)
		t.Fail()
	}
	assert.NotEqual(res, 0)
}
