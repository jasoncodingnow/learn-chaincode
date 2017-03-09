package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"strconv"

	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

//////////////////////////////////////////////////////////////////////////////////////////////////
// The recType is a mandatory attribute. The original app was written with a single table
// in mind. The only way to know how to process a record was the 70's style 80 column punch card
// which used a record type field. The array below holds a list of valid record types.
// This could be stored on a blockchain table or an application
//////////////////////////////////////////////////////////////////////////////////////////////////
var recType = []string{"ARTINV", "USER", "BID", "AUCREQ", "POSTTRAN", "OPENAUC", "CLAUC", "XFER", "VERIFY"}

//////////////////////////////////////////////////////////////////////////////////////////////////
// The following array holds the list of tables that should be created
// The deploy/init deletes the tables and recreates them every time a deploy is invoked
//////////////////////////////////////////////////////////////////////////////////////////////////
var aucTables = []string{"UserTable", "UserCatTable", "ItemTable", "ItemCatTable", "ItemHistoryTable", "AuctionTable", "AucInitTable", "AucOpenTable", "BidTable", "TransTable"}

/////////////////////////////////////////////////////////////////////////////////////////////////////
// A Map that holds TableNames and the number of Keys
// This information is used to dynamically Create, Update
// Replace , and Query the Ledger
// In this model all attributes in a table are strings
// The chain code does both validation
// A dummy key like 2016 in some cases is used for a query to get all rows
//
//              "UserTable":        1, Key: UserID
//              "ItemTable":        1, Key: ItemID
//              "UserCatTable":     3, Key: "2016", UserType, UserID
//              "ItemCatTable":     3, Key: "2016", ItemSubject, ItemID
//              "AuctionTable":     1, Key: AuctionID
//              "AucInitTable":     2, Key: Year, AuctionID
//              "AucOpenTable":     2, Key: Year, AuctionID
//              "TransTable":       2, Key: AuctionID, ItemID
//              "BidTable":         2, Key: AuctionID, BidNo
//              "ItemHistoryTable": 4, Key: ItemID, Status, AuctionHouseID(if applicable),date-time
//
/////////////////////////////////////////////////////////////////////////////////////////////////////
func GetNumberOfKeys(tname string) int {
	TableMap := map[string]int{
		"UserTable":        1,
		"ItemTable":        1,
		"UserCatTable":     3,
		"ItemCatTable":     3,
		"AuctionTable":     1,
		"AucInitTable":     2,
		"AucOpenTable":     2,
		"TransTable":       2,
		"BidTable":         2,
		"ItemHistoryTable": 4,
	}
	return TableMap[tname]
}

// userObject
type UserObject struct {
	UserID    string
	RecType   string // Type = USER
	Name      string
	UserType  string // Auction House (AH), Bank (BK), Buyer or Seller (TR), Shipper (SH), Appraiser (AP)
	Address   string
	Phone     string
	Email     string
	Bank      string
	AccountNo string
	RoutingNo string
}

//////////////////////////////////////////////////////////////
// Invoke Functions based on Function name
// The function name gets resolved to one of the following calls
// during an invoke
//
//////////////////////////////////////////////////////////////
func InvokeFunction(fname string) func(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	InvokeFunc := map[string]func(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error){
		"PostUser": PostUser,
	}
	return InvokeFunc[fname]
}

//////////////////////////////////////////////////////////////
// Query Functions based on Function name
//
//////////////////////////////////////////////////////////////
func QueryFunction(fname string) func(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	QueryFunc := map[string]func(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error){
		"GetUser": GetUser,
	}
	return QueryFunc[fname]
}

// auction chaincode
type SimpleChaincode struct {
}

var gopath string
var ccpath string

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Println("Starting Item Auction Application chaincode")
	gopath = os.Getenv("GOPATH")
	ccpath = fmt.Sprintf("%s/src/github.com/jasoncodingnow/learn-chaincode/auction/", gopath)
	// Start the shim -- running the fabric
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Println("Error starting Item Fun Application chaincode: %s", err)
	}
}

// init function
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("[Trade and Auction Application] Init")
	var err error
	for _, aucTable := range aucTables {
		err = stub.DeleteTable(aucTable)
		if err != nil {
			return nil, fmt.Errorf("Init(), Delete table of %s failed", aucTable)
		}
		err = InitLedger(stub, aucTable)
		if err != nil {
			return nil, fmt.Errorf("Init(): InitLedger of %s  Failed ", aucTable)
		}
	}
	err = stub.PutState("version", []byte(strconv.Itoa(1)))
	if err != nil {
		return nil, err
	}
	fmt.Println("Init() Initialization Complete  : ", args)
	return []byte("Init(): Initialization Complete"), nil
}

// invoke function
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	var err error
	var result []byte

	if ChkReqType(args) == true {
		invokeFunction := InvokeFunction(function)
		if invokeFunction != nil {
			result, err = invokeFunction(stub, function, args)
		}
	} else {
		fmt.Println("Invoke() Invalid recType : ", args, "\n")
		return nil, errors.New("Invoke() : Invalid recType : " + args[0])
	}
	return result, err
}

// Query function
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	var err error
	var result []byte

	fmt.Println("ID Extracted and Type = ", args[0])
	fmt.Println("Args supplied : ", args)

	if len(args) < 1 {
		fmt.Println("Query() : Include at least 1 arguments Key ")
		return nil, errors.New("Query() : Expecting Transation type and Key value for query")
	}

	queryFunction := QueryFunction(function)
	if queryFunction != nil {
		result, err = queryFunction(stub, function, args)
	} else {
		fmt.Println("Query() Invalid function call : ", function)
		return nil, errors.New("Query() : Invalid function call : " + function)
	}
	if err != nil {
		fmt.Println("Query() Object not found : ", args[0])
		return nil, errors.New("Query() : Object not found : " + args[0])
	}
	return result, err
}

// type should in recType
func ChkReqType(args []string) bool {
	for _, rt := range args {
		for _, val := range recType {
			if val == rt {
				return true
			}
		}
	}
	return false
}

// Post User
func PostUser(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	user, err := CreateUserObject(args[0:])
	if err != nil {
		return nil, err
	}
	userbyte, err := UserToJson(user)
	if err != nil {
		fmt.Println("PostuserObject() : Failed Cannot create object buffer for write : ", args[1])
		return nil, errors.New("PostUser(): Failed Cannot create object buffer for write : " + args[1])
	} else {
		keys := []string{args[0]}
		err = UpdateLedger(stub, "UserTable", keys, userbyte)
		if err != nil {
			fmt.Println("PostUser() : write error while inserting record")
			return nil, err
		}
		keys = []string{"2017", args[3], args[0]}
		err = UpdateLedger(stub, "UserCatTable", keys, userbyte)
		if err != nil {
			fmt.Println("PostUser() : write error while inserting recordinto UserCatTable \n")
			return nil, err
		}
	}
	return userbyte, err
}

func GetUser(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	var err error
	databytes, err := QueryFromLedger(stub, "UserTable", args)
	if err != nil {
		fmt.Println("GetUser() : Failed to Query Object ")
		jsonResp := "{\"Error\":\"Failed to get  Object Data for " + args[0] + "\"}"
		return nil, errors.New(jsonResp)
	}
	if databytes == nil {
		fmt.Println("GetUser() : Incomplete Query Object ")
		jsonResp := "{\"Error\":\"Incomplete information about the key for " + args[0] + "\"}"
		return nil, errors.New(jsonResp)
	}

	fmt.Println("GetUser() : Response : Successfull -")
	return databytes, nil
}

// create user from args
func CreateUserObject(args []string) (UserObject, error) {
	var err error
	var user UserObject

	if len(args) != 10 {
		fmt.Println("CreateUserObject(): Incorrect number of arguments. Expecting 10 ")
		return user, errors.New("CreateUserObject() : Incorrect number of arguments. Expecting 10 ")
	}

	_, err = strconv.Atoi(args[0])
	if err != nil {
		return user, errors.New("CreateUserObject() : User ID should be an integer")
	}
	user = UserObject{args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9]}
	fmt.Println("CreateUserObject() : User Object : ", user)
	return user, err
}

// convert UserObject to json
func UserToJson(user UserObject) ([]byte, error) {
	userjson, err := json.Marshal(user)
	if err != nil {
		fmt.Println("UserToJson err:", err)
		return nil, err
	}
	return userjson, nil
}

// Ledger function

// init ledger
func InitLedger(stub shim.ChaincodeStubInterface, tableName string) error {
	nKeys := GetNumberOfKeys(tableName)
	if nKeys < 1 {
		fmt.Println("At least 1 Key must be provided \n")
		fmt.Println("Auction_Application: Failed creating Table ", tableName)
		return errors.New("Auction_Application: Failed creating Table " + tableName)
	}

	var colDefForTbl []*shim.ColumnDefinition
	for i := 0; i < nKeys; i++ {
		columnDef := shim.ColumnDefinition{Name: "keyName" + strconv.Itoa(i), Type: shim.ColumnDefinition_STRING, Key: true}
		colDefForTbl = append(colDefForTbl, &columnDef)
	}

	colLastDef := shim.ColumnDefinition{Name: "Details", Type: shim.ColumnDefinition_BYTES, Key: false}
	colDefForTbl = append(colDefForTbl, &colLastDef)

	err := stub.CreateTable(tableName, colDefForTbl)
	if err != nil {
		fmt.Println("Auction_Application: Failed creating Table ", tableName)
		return errors.New("Auction_Application: Failed creating Table " + tableName)
	}
	return err
}

func UpdateLedger(stub shim.ChaincodeStubInterface, tableName string, key []string, args []byte) error {
	nKeys := GetNumberOfKeys(tableName)
	if nKeys < 1 {
		fmt.Println("Atleast 1 Key must be provided \n")
	}
	var columns []*shim.Column

	for i := 0; i < nKeys; i++ {
		col := shim.Column{Value: &shim.Column_String_{String_: key[i]}}
		columns = append(columns, &col)
	}
	lastCol := shim.Column{Value: &shim.Column_Bytes{Bytes: []byte(args)}}
	columns = append(columns, &lastCol)

	currRow := shim.Row{columns}
	ok, err := stub.InsertRow(tableName, currRow)
	if err != nil {
		return fmt.Errorf("UpdateLedger: InsertRow into "+tableName+" Table operation failed. %s", err)
	}
	if !ok {
		return errors.New("UpdateLedger: InsertRow into " + tableName + " Table failed. Row with given key " + key[0] + " already exists")
	}

	fmt.Println("UpdateLedger: InsertRow into ", tableName, " Table operation Successful. ")
	return nil
}

// DeleteFromLedger
func DeleteFromLedger(stub shim.ChaincodeStubInterface, tableName string, keys []string) error {
	var columns []shim.Column

	nCol := len(keys)
	if nCol < 1 {
		fmt.Println("Atleast 1 Key must be provided \n")
		return errors.New("DeleteFromLedger failed. Must include at least key values")
	}

	for i := 0; i < nCol; i++ {
		col := shim.Column{Value: &shim.Column_String_{String_: keys[i]}}
		columns = append(columns, col)
	}

	err := stub.DeleteRow(tableName, columns)
	if err != nil {
		return fmt.Errorf("DeleteFromLedger operation failed. %s", err)
	}

	fmt.Println("DeleteFromLedger: DeleteRow from ", tableName, " Table operation Successful. ")
	return nil
}

// ReplaceRowFromLedger
func ReplaceRowFromLedger(stub shim.ChaincodeStubInterface, tableName string, keys []string, data []byte) error {
	nKeys := GetNumberOfKeys(tableName)
	if nKeys < 1 {
		fmt.Println("Atleast 1 Key must be provided \n")
	}

	var columns []*shim.Column
	for i := 0; i < nKeys; i++ {
		col := shim.Column{Value: &shim.Column_String_{String_: keys[i]}}
		columns = append(columns, &col)
	}

	dataCol := shim.Column{Value: &shim.Column_Bytes{Bytes: data}}
	columns = append(columns, &dataCol)

	row := shim.Row{columns}
	ok, err := stub.ReplaceRow(tableName, row)
	if err != nil {
		return fmt.Errorf("ReplaceLedgerEntry: Replace Row into "+tableName+" Table operation failed. %s", err)
	}
	if !ok {
		return errors.New("ReplaceLedgerEntry: Replace Row into " + tableName + " Table failed. Row with given key " + keys[0] + " already exists")
	}

	fmt.Println("ReplaceLedgerEntry: Replace Row in ", tableName, " Table operation Successful. ")
	return nil
}

// QueryFromLedger
func QueryFromLedger(stub shim.ChaincodeStubInterface, tableName string, args []string) ([]byte, error) {
	var columns []shim.Column
	nKeys := GetNumberOfKeys(tableName)
	for i := 0; i < nKeys; i++ {
		col := shim.Column{Value: &shim.Column_String_{String_: args[i]}}
		columns = append(columns, col)
	}
	row, err := stub.GetRow(tableName, columns)
	fmt.Println("Length or number of rows retrieved ", len(row.Columns))

	if len(row.Columns) == 0 {
		jsonResp := "{\"Error\":\"Failed retrieving data " + args[0] + ". \"}"
		fmt.Println("Error retrieving data record for Key = ", args[0], "Error : ", jsonResp)
		return nil, errors.New(jsonResp)
	}
	// 所有的数据, key 之后的都是 RecType
	dataBytes := row.Columns[nKeys].GetBytes()
	err = ProcessQueryResult(stub, dataBytes, args)
	if err != nil {
		fmt.Println("QueryLedger() : Cannot create object  : ", args[1])
		jsonResp := "{\"QueryLedger() Error\":\" Cannot create Object for key " + args[0] + "\"}"
		return nil, errors.New(jsonResp)
	}
	return dataBytes, nil
}

// ProcessQueryResult: check if correct
func ProcessQueryResult(stub shim.ChaincodeStubInterface, bytes []byte, args []string) error {
	var dat map[string]interface{}
	if err := json.Unmarshal(bytes, &dat); err != nil {
		panic(err)
	}

	var recType string
	recType = dat["RecTYpe"].(string)
	switch recType {
	case "USER":
		_, err := JsonToUser(bytes)
		if err != nil {
			return err
		}
		return err
	}
	return nil
}

func JsonToUser(bytes []byte) (UserObject, error) {
	user := UserObject{}
	err := json.Unmarshal(bytes, &user)
	if err != nil {
		fmt.Println("JSONtoUser error: ", err)
		return user, err
	}
	fmt.Println("JSONtoUser created: ", user)
	return user, err
}
