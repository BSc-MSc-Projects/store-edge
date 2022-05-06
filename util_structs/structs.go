package utilstructs

/* Simple socket struct used server side */
type Socket struct {
	Ip_addr string
	Port    int
}

/* Tells the status associated to an RPC, embedded in the Record struct to check client side if everything was ok */
type StatusInfo struct {
}

/* Record struct, used both client and server side to save and retrieve key:[]value elements*/
type Record struct {
	key        string
	value      []string
	ttl        float32
	timeUpdate float32
	amIowner   bool
	ownerAddr  Socket
	totalSize  int32
}

func SetKey(r *Record, key string) {
	r.key = key
}

func AppendValue(r *Record, value string) {
	r.value = append(r.value, value)
}

func PutValue(r *Record, value []string) {
	r.value = value
}

func SetTtl(r *Record, ttl float32) {
	r.ttl = ttl
}

func SetOwner(r *Record, owner bool, ownerAddr Socket) {
	r.amIowner = owner
	if (ownerAddr.Ip_addr != "") && (ownerAddr.Port != 0) {
		r.ownerAddr = ownerAddr
	}
}

func IncrementTotalSize(r *Record, length int32) int32 {
	r.totalSize += length
	return r.totalSize
}

func RemoveAll(r *Record) {
	r.value = r.value[:0] // delete all the elements
	r.totalSize = 0
}

func GetKey(r *Record) string {
	return r.key
}

func GetValues(r *Record) []string {
	return r.value
}

func GetTtl(r *Record) float32 {
	return r.ttl
}

func GetOwner(r *Record) (bool, Socket) {
	return r.amIowner, r.ownerAddr
}

func GetTotalSize(r *Record) int32 {
	return r.totalSize
}

func DecrementTTL(r *Record, decTtl float32) {
	r.ttl = r.ttl - decTtl
}

func IncreaseTimeSinceLastUpdate(r *Record, passedTime float32) {
	r.timeUpdate = r.timeUpdate + passedTime
}

func SetTimeSinceLastUpdate(r *Record, tslu float32) {
	r.timeUpdate = tslu
}

func GetTimeSinceLastUpdate(r *Record) float32 {
	return r.timeUpdate
}

/* Dynamo object, saved in an appropriate DynamoDB table*/
type DynamoObject struct {
	Key  string
	Vals []string `dynamodbav:"vals,omitempty"`
	Type int32
}
