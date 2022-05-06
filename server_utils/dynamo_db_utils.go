/* Dynamo DB Golang API, to interact with the S3 service. To make things work, the API makes the subsequent assumptions:
 * - you have stored in your local machine 2 files, under the .aws/ folder in your home directory
 *		1) credentials: this file contains credentials to log into your AWS account
 *		2) config: this file contains the configuration for the account, for instance the AWS region and desired output format
 *		for the operations
 * - there is a table created on DynamoDB called "HeavyValues"
 *
 * The API allows to:
 * 		- create a new voice on the DynamoDB table
 *		- retrieve a voice corresponding to a given key from the table
 *		- append a value to an existing voice
 *		- delete a voice
 * The API integrates the storage on the edge node, to save those values who are either heavy or not frequently accessed
 */

package server_utils

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"

	"log"

	structs "github.com/piercirocaliandro/sdcc-edge-computing/util_structs"
)

/* Upload an element on a DynamoDB table
 *
 * @param putItem: pointer to a DynamoObject type, containing all the information to save the record on DynamoDB
 */
func DynamoDBPutObject(putItem *structs.DynamoObject) {

	svc := dynamoSessCreate()
	av, err := dynamodbattribute.MarshalMap(putItem)
	if err != nil {
		log.Fatalf("Got error marshalling new movie item: %s", err)
	}

	// Create item in table Movies
	tableName := "HeavyValues"

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(tableName),
	}

	_, err = svc.PutItem(input)
	if err != nil {
		log.Fatalf("Got error calling PutItem: %s", err)
	}
}

/* Retrieve elements from DynamoDB
 *
 * @param retrItem: DynamoObject struct, containing all the information needed to retrieve the object from AWS
 */
func DynamoDBRetrieveObjects(retrItem *structs.DynamoObject) *structs.Record {
	svc := dynamoSessCreate()
	tableName := "HeavyValues"
	key := retrItem.Key

	// fmt.Printf("Trying to fetch items with key: %v\n", retrItem.Key)

	// Create the Expression to fill the input struct with.
	// Get all the values associated with the specified key
	filt := expression.Name("Key").Equal(expression.Value(key))

	// Get back the key, value and valuetype fields
	proj := expression.NamesList(expression.Name("Key"), expression.Name("vals"), expression.Name("ValueType"))

	expr, err := expression.NewBuilder().WithFilter(filt).WithProjection(proj).Build()
	if err != nil {
		log.Fatalf("Got error building expression: %s", err)
	}
	// Build the query input parameters
	params := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String(tableName),
	}

	// Make the DynamoDB Query API call
	result, err := svc.Scan(params)
	if err != nil {
		log.Fatalf("Query API call failed: %s", err)
	}

	/* Create a new Record struct, that will be returned and sent to the client */
	retRecord := structs.Record{}
	for _, i := range result.Items {
		item := structs.DynamoObject{}

		err = dynamodbattribute.UnmarshalMap(i, &item)
		if err != nil {
			log.Fatalf("Got error unmarshalling: %s", err)
		}
		structs.SetKey(&retRecord, item.Key)
		for _, value := range item.Vals {
			structs.AppendValue(&retRecord, value)
		}
	}
	return &retRecord
}

/* Delete an Item from DynamoDB
 * @param key: the key of the item to delete
 */
func DynamoDBDeleteObjects(key string) int {
	svc := dynamoSessCreate()
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"Key": {
				S: &key,
			},
		},
		TableName: aws.String("HeavyValues"),
	}

	_, err := svc.DeleteItem(input)
	if err != nil {
		log.Printf("%v\n", err.Error())
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			return 0
		}
		return -1
	}
	return 0
}

/* Append a value to a specific key, if present in DynamoDB
 * @param key: the key for the element
 * @parm value: the value to append
 */
func DynamoDBAppendValue(key string, value string) int {
	svc := dynamoSessCreate()

	av := &dynamodb.AttributeValue{
		S: aws.String(value),
	}

	// fmt.Println("Trying to update for the element with key:", key)

	var newVals []*dynamodb.AttributeValue
	newVals = append(newVals, av)

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v": {
				L: newVals,
			},
		},
		ExpressionAttributeNames: map[string]*string{
			"#attrName": aws.String("vals"),
		},
		TableName: aws.String("HeavyValues"),
		Key: map[string]*dynamodb.AttributeValue{
			"Key": {
				S: aws.String(key),
			},
		},
		UpdateExpression: aws.String("SET #attrName = list_append(#attrName, :v)"),
	}
	_, err := svc.UpdateItem(input)
	if err != nil {
		log.Printf("Got error calling UpdateItem: %s", err)
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			return 0
		}
		return -1
	}
	// fmt.Println("Append on Dynamo successfully completed")
	return 0
}

/* ------------------------------- Private functions, used for set up -------------------------------------------------------- */

/* Create a new ASW session to use golang SDK for DynamoDB. The default files that are needed:
 *	- credentials from the shared credentials file ~/.aws/credentials
 *	- and region from the shared configuration file ~/.aws/config.
 */
func dynamoSessCreate() *dynamodb.DynamoDB {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Create DynamoDB client and return it
	return dynamodb.New(sess)
}
