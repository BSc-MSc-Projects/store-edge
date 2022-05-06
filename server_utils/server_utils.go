/* This file contains all the utility functions needed to inerface with the AWS S3 Golang SDK
*  in particular, it offers an API to:
*	- list all the buckets in a given region (the region is for default us-east-1
*	- put a new object on a bucket
* 	- get an object from a bucketà
*	- append an object to a bucket: to do so, files in buckets are stored in a directory, so the key of the object is given by
	  <user-key>/filename.ext
*	- delete all the elements in a bucket that contains a specific prefix
*
*  The application deals only with textual values, so even if they are saved as files, they will be returned as []byte
*
* TODO: delete this file, it is not used anymore
*/

package server_utils

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	structs "github.com/piercirocaliandro/sdcc-edge-computing/util_structs"
)

/* Set up the S3 sdk, connecting to the S3 bucket in the correct region */
func SetUpS3() *session.Session {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	return sess // return the created session
}

// List the Buckets presents on S3
func ListBuckets(sess *session.Session) *s3.Bucket {
	svc := s3.New(sess, &aws.Config{
		Region: aws.String(endpoints.UsEast1RegionID),
	})
	result, err := svc.ListBuckets(nil)

	if err != nil {
		log.Fatalf("An error occurred: %v\n", err)
	}
	return result.Buckets[0] // always return bucket 0 because for now there is only a single bucket
}

/* Retrieves all the objects associated to a specific S3 bucket. After this, deletes
 * all the Objects which key contains the string specfied by "keyPrefix"

 * @param keyPrefix: the key which values need to be deleted
 * @param sess: S3 client session
 * @param bucket: S3 bucket Object
 */
func BatchDeleteS3Objects(keyPrefix string, sess *session.Session, bucket s3.Bucket) {
	svc := s3.New(sess, &aws.Config{
		Region: aws.String(endpoints.UsEast1RegionID),
	})

	// List the object contained on a Bucket
	input := &s3.ListObjectsInput{
		Bucket: bucket.Name,
	}
	result, err := svc.ListObjects(input)
	if err != nil {
		log.Fatalf("Error occurred while retrieving objects: %v\n", err)
	}

	if len(result.Contents) == 0 {
		return
	}
	delObjects := []*s3.ObjectIdentifier{}

	for _, object := range result.Contents {
		if strings.Contains(*object.Key, keyPrefix) {
			delObjects = append(delObjects, &s3.ObjectIdentifier{
				Key: object.Key,
			})
		}
	}

	deleteInput := &s3.DeleteObjectsInput{
		Bucket: bucket.Name,
		Delete: &s3.Delete{
			Objects: delObjects,
		},
	}
	delResult, delErr := svc.DeleteObjects(deleteInput)
	if delErr != nil {
		log.Fatalf("Error occurred during deletetition %v\n", delErr)
	}
	fmt.Println("Successfully deleted objects in :", delResult.Deleted)
}

/* Upload a value to an S3 bucket. We use S3 since it is easier to save and fetch values, however we do not handle data types
 * different from []byte and so, each time that we save a new record, we just pass the value as raw data
 * @param sess: the client session to contact S3
 * @param filename: the name of the file to upload
 * @param key: the user-specified key associated to the file name
 * @param bucket: S3 bucket Object where to put the new file
 * @param serverPath: the server path needed to load the file
 * @param erase: if false, the put is in append mode
 */
func UploadToS3(sess *session.Session, value string, key string, bucket s3.Bucket, serverPath string, erase bool) {
	if erase {
		BatchDeleteS3Objects(key, sess, bucket)
	}

	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)

	// Upload the file to S3.
	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(*bucket.Name),
		Key:    aws.String(key + "/" + value), // to exploit S3 directory organization, the actual key is ket + filename
	})
	if err != nil {
		log.Fatalf("failed to upload file, %v", err)
	}
	fmt.Printf("file uploaded to %s\n", result.Location)
}

/* Get a file from the specified S3 bucket
 * @param sess: session opened to S3
 * @param filename: the name of the file to retrieve
 * @param key: the key associated to the string
 */
func DownloadFromS3(sess *session.Session, filename string, key string, bucket s3.Bucket, serverDir string) *structs.Record {
	fmt.Printf("Here I am, trying to go on S3 for the key %v\n", key)
	// Create a downloader with the session and default options
	downloader := s3manager.NewDownloader(sess)

	/* A key could have multiple files associated, list the Object on S3 and retrieve them */
	svc := s3.New(sess, &aws.Config{
		Region: aws.String(endpoints.UsEast1RegionID),
	})
	// List the object contained on a Bucket
	input := &s3.ListObjectsInput{
		Bucket: bucket.Name,
	}
	result, err := svc.ListObjects(input)
	if err != nil {
		log.Fatalf("Error occurred while retrieving objects: %v\n", err)
	}

	if len(result.Contents) == 0 {
		emptyRecord := structs.Record{}
		structs.SetKey(&emptyRecord, "")
		return &emptyRecord
	}

	fetchRecord := structs.Record{}
	for _, object := range result.Contents {
		if strings.Contains(*object.Key, key) {
			objFilename := getFileNameFromS3Key(*object.Key)
			// Create a file to write the S3 Object contents to.
			f, err := os.Create(serverDir + "/" + objFilename)
			if err != nil {
				log.Fatalf("failed to create file %q, %v", objFilename, err)
				retWithError("exiting...")
			}
			fmt.Println("File created: ", objFilename)
			// Write the contents of S3 Object to the file
			n, err := downloader.Download(f, &s3.GetObjectInput{
				Bucket: aws.String(*bucket.Name),
				Key:    aws.String(*object.Key),
			})
			if err != nil {
				retWithError("failed to download file\n")
			}
			fmt.Printf("file downloaded, %d bytes\n", n)
			structs.SetKey(&fetchRecord, key)
			structs.AppendValue(&fetchRecord, objFilename)
		}
	}
	return &fetchRecord
}

func retWithError(err string) *structs.Record {
	log.Fatalf("%s", err)
	errRecord := structs.Record{}
	structs.SetKey(&errRecord, "")
	return &errRecord
}

func getFileNameFromS3Key(s3Key string) string {
	var filename []byte
	for index := len(s3Key) - 1; s3Key[index] != '/'; index-- {
		filename = append(filename, s3Key[index])
	}
	return reverseString(string(filename))
}

func reverseString(str string) string {
	byte_str := []rune(str)
	for i, j := 0, len(byte_str)-1; i < j; i, j = i+1, j-1 {
		byte_str[i], byte_str[j] = byte_str[j], byte_str[i]
	}
	return string(byte_str)
}
