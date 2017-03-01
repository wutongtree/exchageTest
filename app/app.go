package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"math"

	"github.com/gocraft/web"
	"github.com/hyperledger/fabric/core/util"
)

// restResult defines the response payload for a general REST interface request.
type restResult struct {
	OK  interface{}
	Err string
}

type restResp struct {
	Status string      `json:"status"`
	Result interface{} `json:"result"`
}

type respErr struct {
	Code string `json:"code"`
	Msg  string `json:"mag"`
}

type User struct {
	EnrollID     string `json:"enrollId"`
	EnrollSecret string `json:"enrollSecret"`
}

const (
	SUCCESS  = "success"
	FAILED   = "failed"
	SYSERR   = "SYS_ERR"
	NOTLOGIN = "NOT_LOGIN"
	PARAMERR = "REQ_PARAM_ERR"
)

// Order Order
type Order struct {
	UUID         string  `json:"uuid"`        //UUID
	Account      string  `json:"account"`     //账户
	SrcCurrency  string  `json:"srcCurrency"` //源币种代码
	SrcCount     float64 `json:"srcCount"`    //源币种交易数量
	DesCurrency  string  `json:"desCurrency"` //目标币种代码
	DesCount     float64 `json:"desCount"`    //目标币种交易数量
	IsBuyAll     bool    `json:"isBuyAll"`    //是否买入所有，即为true是以目标币全部兑完为主,否则算部分成交,买完为止；为false则是以源币全部兑完为主,否则算部分成交，卖完为止
	ExpiredTime  int64   `json:"expiredTime"` //超时时间
	ExpiredDate  string  `json:"expiredDate"`
	PendingTime  int64   `json:"PendingTime"` //挂单时间
	PendingDate  string  `json:"pendingDate"`
	PendedTime   int64   `json:"PendedTime"` //挂单完成时间
	PendedDate   string  `json:"pendedDate"`
	MatchedTime  int64   `json:"matchedTime"` //撮合完成时间
	MatchedDate  string  `json:"matchedDate"`
	FinishedTime int64   `json:"finishedTime"` //交易完成时间
	FinishedDate string  `json:"finishedDate"`
	RawUUID      string  `json:"rawUUID"`   //母单UUID
	Metadata     string  `json:"metadata"`  //存放其他数据，如挂单锁定失败信息
	FinalCost    float64 `json:"finalCost"` //源币的最终消耗数量，主要用于买完（IsBuyAll=true）的最后一笔交易计算结余，此时SrcCount有可能大于FinalCost
	Status       int     `json:"status"`    //状态 0：待交易，1：完成，2：过期，3：撤单
}

// Order Order
type OrderInt struct {
	UUID         string `json:"uuid"`         //UUID
	Account      string `json:"account"`      //账户
	SrcCurrency  string `json:"srcCurrency"`  //源币种代码
	SrcCount     int64  `json:"srcCount"`     //源币种交易数量
	DesCurrency  string `json:"desCurrency"`  //目标币种代码
	DesCount     int64  `json:"desCount"`     //目标币种交易数量
	IsBuyAll     bool   `json:"isBuyAll"`     //是否买入所有，即为true是以目标币全部兑完为主,否则算部分成交,买完为止；为false则是以源币全部兑完为主,否则算部分成交，卖完为止
	ExpiredTime  int64  `json:"expiredTime"`  //超时时间
	PendingTime  int64  `json:"PendingTime"`  //挂单时间
	PendedTime   int64  `json:"PendedTime"`   //挂单完成时间
	MatchedTime  int64  `json:"matchedTime"`  //撮合完成时间
	FinishedTime int64  `json:"finishedTime"` //交易完成时间
	RawUUID      string `json:"rawUUID"`      //母单UUID
	Metadata     string `json:"metadata"`     //存放其他数据，如挂单锁定失败信息
	FinalCost    int64  `json:"finalCost"`    //源币的最终消耗数量，主要用于买完（IsBuyAll=true）的最后一笔交易计算结余，此时SrcCount有可能大于FinalCost
	Status       int    `json:"status"`       //状态
}

// NotFound NotFound
func (a *AppREST) NotFound(rw web.ResponseWriter, req *web.Request) {
	rw.WriteHeader(http.StatusNotFound)
	json.NewEncoder(rw).Encode(restResult{Err: "Request not found."})
}

// SetResponseType is a middleware function that sets the appropriate response
// headers. Currently, it is setting the "Content-Type" to "application/json" as
// well as the necessary headers in order to enable CORS for Swagger usage.
func (s *AppREST) SetResponseType(rw web.ResponseWriter, req *web.Request, next web.NextMiddlewareFunc) {
	rw.Header().Set("Content-Type", "application/json")

	// Enable CORS
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Access-Control-Allow-Headers", "accept, content-type")

	next(rw, req)
}

type Currency struct {
	ID         string  `json:"id"`
	Count      float64 `json:"count"`
	LeftCount  float64 `json:"leftCount"`
	Creator    string  `json:"creator"`
	User       string  `json:"user"`
	CreateTime int64   `json:"createTime"`
}

var Multiple = math.Pow10(6)

// Create 创建币
func (a *AppREST) Create(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Info("REST processing create currency request...")

	encoder := json.NewEncoder(rw)

	// Read in the incoming request payload
	reqBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: PARAMERR, Msg: "Internal JSON error when reading request body"}})
		// myLogger.Error("Internal JSON error when reading request body.")
		return
	}

	// Incoming request body may not be empty, client must supply request payload
	if string(reqBody) == "" {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: PARAMERR, Msg: "Client must supply a payload for order requests"}})
		// myLogger.Error("Client must supply a payload for order requests.")
		return
	}
	// myLogger.Debugf("createCurrency request body :%s", string(reqBody))

	// Payload must conform to the following structure
	var currency Currency

	// Decode the request payload as an Request structure.	There will be an
	// error here if the incoming JSON is invalid
	err = json.Unmarshal(reqBody, &currency)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: PARAMERR, Msg: "request parameter is wrong"}})
		// myLogger.Errorf("Error unmarshalling order request payload: %s", err)
		return
	}

	// 校验请求数据
	if len(currency.ID) <= 0 {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: PARAMERR, Msg: "Currency cann't be empty"}})
		// myLogger.Error("Currency cann't be empty.")
		return
	}
	if currency.Count < 0 {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: PARAMERR, Msg: "Count must be greater than 0"}})
		// myLogger.Error("Count must be greater than 0.")
		return
	}

	// chaincode
	txid, err := createCurrency(currency.ID, int64(round(currency.Count, 6)*Multiple), currency.User)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: SYSERR, Msg: "create Currency failed"}})
		// myLogger.Errorf("create Currency failed:%s", err)
		return
	}

	rw.WriteHeader(http.StatusOK)
	encoder.Encode(restResp{Status: SUCCESS, Result: struct{ Txid string }{Txid: txid}})
}

// CheckCreate  检测创建币结果，由前端轮询
// response说明：StatusBadRequest  失败  不需继续轮询，Error表示失败原因
//				StatusOK OK="1" 成功  不需继续轮询
//				StatusOK OK="0" 未果  需要继续轮询
func (a *AppREST) CheckCreate(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Info("REST processing check create request...")

	encoder := json.NewEncoder(rw)

	txid := req.PathParams["txid"]
	if txid == "" {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: PARAMERR, Msg: "Client must supply a id for checkcreate requests"}})
		// myLogger.Errorf("Client must supply a id for checkcreate requests.")
		return
	}
	// myLogger.Debugf("check create request parameter:txid = %s", txid)

	v, ok := chaincodeResult[txid]
	if !ok {
		rw.WriteHeader(http.StatusOK)
		encoder.Encode(restResult{OK: "0"})
	} else if v == Chaincode_Success {
		rw.WriteHeader(http.StatusOK)
		encoder.Encode(restResult{OK: "1"})
	} else {
		encoder.Encode(restResult{Err: v})
	}
}

// Currency 获取币信息
func (a *AppREST) Currency(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Info("REST processing get currency request...")

	encoder := json.NewEncoder(rw)

	id := req.PathParams["id"]
	if id == "" {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: PARAMERR, Msg: "Currency id can't be empty"}})
		// myLogger.Error("Get currency failed:Currency id can't be empty")
		return
	}
	// myLogger.Debugf("Get currency parameter id = %s", id)

	result, err := getCurrency(id)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: SYSERR, Msg: "Get currency failed"}})
		// myLogger.Errorf("Get currency failed:%s", err)
		return
	}

	var currency Currency
	err = json.Unmarshal([]byte(result), &currency)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: SYSERR, Msg: "Get currency failed"}})
		// myLogger.Errorf("Get currency failed:%s", err)
		return
	}

	currency.Count = currency.Count / Multiple
	currency.LeftCount = currency.LeftCount / Multiple

	rw.WriteHeader(http.StatusOK)
	encoder.Encode(restResp{Status: SUCCESS, Result: struct{ Currency }{currency}})
}

// Currencys 获取币信息
func (a *AppREST) Currencys(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Info("REST processing get all currency request...")

	encoder := json.NewEncoder(rw)

	result, err := getCurrencys()
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: SYSERR, Msg: "Get currency failed"}})
		// myLogger.Errorf("Get currency failed: %s", err)
		return
	}

	var currencys []Currency
	err = json.Unmarshal([]byte(result), &currencys)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: SYSERR, Msg: "Get currency failed"}})
		// myLogger.Errorf("Get currency failed : %s", err)
		return
	}

	for k, v := range currencys {
		currencys[k].Count = v.Count / Multiple
		currencys[k].LeftCount = v.LeftCount / Multiple
	}

	rw.WriteHeader(http.StatusOK)
	encoder.Encode(restResp{Status: SUCCESS, Result: struct{ Currencys []Currency }{Currencys: currencys}})
}

type Asset struct {
	Owner     string  `json:"owner"`
	Currency  string  `json:"currency"`
	Count     float64 `json:"count"`
	LockCount float64 `json:"lockCount"`
}

// Asset 获取个人资产信息
func (a *AppREST) Asset(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Info("REST processing get currency request...")

	encoder := json.NewEncoder(rw)

	owner := req.PathParams["owner"]
	if owner == "" {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "owner can't be empty"})
		// myLogger.Errorf("Get asset failed")
		return
	}

	result, err := getAsset(owner)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Get owner asset failed"})
		// myLogger.Errorf("Get owner asset failed")
		return
	}

	var infos []Asset
	err = json.Unmarshal([]byte(result), &infos)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Get owner asset failed"})
		// myLogger.Errorf("Get owner asset failed")
		return
	}

	for k, v := range infos {
		infos[k].Count = v.Count / Multiple
		infos[k].LockCount = v.LockCount / Multiple
	}

	// js, _ := json.Marshal(&infos)
	rw.WriteHeader(http.StatusOK)
	encoder.Encode(restResult{OK: infos})
}

// MyCurrency 个人创建的币
func (a *AppREST) MyCurrency(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Info("REST processing get user currency request...")

	encoder := json.NewEncoder(rw)

	user := req.PathParams["user"]
	if user == "" {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "user can't be empty"})
		// myLogger.Errorf("Get currency failed")
		return
	}

	result, err := getCurrencysByUser(user)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Get currency failed"})
		// myLogger.Errorf("Get currency failed")
		return
	}

	var infos []Currency
	err = json.Unmarshal([]byte(result), &infos)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Get currency failed"})
		// myLogger.Errorf("Get currency failed")
		return
	}

	for k, v := range infos {
		infos[k].Count = v.Count / Multiple
		infos[k].LeftCount = v.LeftCount / Multiple
	}

	// js, _ := json.Marshal(&infos)
	rw.WriteHeader(http.StatusOK)
	encoder.Encode(restResult{OK: infos})
}

// MyTxs 个人挂单记录
func (a AppREST) MyTxs(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Info("REST processing get user txs request...")

	encoder := json.NewEncoder(rw)

	user := req.PathParams["user"]
	if user == "" {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "user can't be empty"})
		// myLogger.Errorf("Get currency failed")
		return
	}

	// 用户挂单集中于挂单成功队列和以用户名为key的队列中 且 不在以下三种状态的 status = 0
	// 完成的挂单存在于交易执行成功队列中 status = 1
	// 过期的挂单存在于过期成功队列中 status = 2
	// 撤单的挂单存在于撤单成功队列中 status = 3
	uuids, err := getAllSetMember("user_" + user)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Get txs failed"})
		// myLogger.Errorf("Get txs failed")
		return
	}

	var txs []Order
	for _, v := range uuids {
		order, _ := getOrder(v)
		if ok, _ := isInSet(ExchangeSuccessKey, v); ok {
			order.Status = 1
		} else if ok, _ := isInSet(ExpiredSuccessOrderKey, v); ok {
			order.Status = 2
		} else if ok, _ := isInSet(CancelSuccessOrderKey, v); ok {
			order.Status = 3
		} else {
			order.Status = 0
		}

		txs = append(txs, *order)
	}

	rw.WriteHeader(http.StatusOK)
	encoder.Encode(restResult{OK: txs})
}

// Release 发布币
func (a *AppREST) Release(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Info("REST processing currency release request...")

	encoder := json.NewEncoder(rw)

	// Read in the incoming request payload
	reqBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		encoder.Encode(restResult{Err: "Internal JSON error when reading request body."})

		// myLogger.Error("Internal JSON error when reading request body.")
		return
	}

	// Incoming request body may not be empty, client must supply request payload
	if string(reqBody) == "" {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Client must supply a payload for order requests."})

		// myLogger.Error("Client must supply a payload for order requests.")
		return
	}

	// Payload must conform to the following structure
	var currency Currency

	// Decode the request payload as an Request structure.	There will be an
	// error here if the incoming JSON is invalid
	err = json.Unmarshal(reqBody, &currency)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: fmt.Sprintf("Error unmarshalling order request payload: %s", err)})

		// myLogger.Errorf("Error unmarshalling order request payload: %s", err)
		return
	}

	// 校验请求数据
	if len(currency.ID) <= 0 {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Currency cann't be empty."})

		// myLogger.Error("Currency cann't be empty.")
		return
	}
	if currency.Count < 0 {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Count must be greater than 0."})

		// myLogger.Error("Count must be greater than 0.")
		return
	}

	// chaincode
	txid, err := releaseCurrency(currency.ID, int64(round(currency.Count, 6)*Multiple), currency.User)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "release Currency failed."})
		return
	}

	rw.WriteHeader(http.StatusOK)
	encoder.Encode(restResult{OK: txid})
}

// CheckRelease 检测发布币结果
// response说明：StatusBadRequest  失败  不需继续轮询，Error表示失败原因
//				StatusOK OK="1" 成功  不需继续轮询
//				StatusOK OK="0" 未果  需要继续轮询
func (a *AppREST) CheckRelease(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Info("REST processing check release request...")

	encoder := json.NewEncoder(rw)

	txid := req.PathParams["txid"]
	if txid == "" {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Client must supply a id for checkrelease requests."})

		// myLogger.Errorf("Client must supply a id for checkrelease requests.")
		return
	}

	v, ok := chaincodeResult[txid]
	if !ok {
		rw.WriteHeader(http.StatusOK)
		encoder.Encode(restResult{OK: "0"})
	} else if v == Chaincode_Success {
		rw.WriteHeader(http.StatusOK)
		encoder.Encode(restResult{OK: "1"})
	} else {
		encoder.Encode(restResult{Err: v})
	}
}

// Assign 分发币
func (a *AppREST) Assign(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Info("REST processing currency assign request...")

	encoder := json.NewEncoder(rw)

	// Read in the incoming request payload
	reqBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		encoder.Encode(restResult{Err: "Internal JSON error when reading request body."})

		// myLogger.Error("Internal JSON error when reading request body.")
		return
	}

	// Incoming request body may not be empty, client must supply request payload
	if string(reqBody) == "" {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Client must supply a payload for order requests."})

		// myLogger.Error("Client must supply a payload for order requests.")
		return
	}

	// Payload must conform to the following structure
	var assign struct {
		User     string `json:"user"`
		Currency string `json:"currency"`
		Assigns  []struct {
			Owner string `json:"owner"`
			Count int64  `json:"count"`
		} `json:"assigns"`
	}

	// Decode the request payload as an Request structure.	There will be an
	// error here if the incoming JSON is invalid
	err = json.Unmarshal(reqBody, &assign)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: fmt.Sprintf("Error unmarshalling order request payload: %s", err)})

		// myLogger.Errorf("Error unmarshalling order request payload: %s", err)
		return
	}

	// 校验请求数据
	if len(assign.Currency) <= 0 {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Currency cann't be empty."})

		// myLogger.Error("Currency cann't be empty.")
		return
	}
	for k, v := range assign.Assigns {
		if v.Count < 0 {
			rw.WriteHeader(http.StatusBadRequest)
			encoder.Encode(restResult{Err: "Count must be greater than 0."})

			// myLogger.Error("Count must be greater than 0.")
			return
		}
		assign.Assigns[k].Count = int64(float64(v.Count) * Multiple)
	}

	assigns, _ := json.Marshal(&assign)
	// chaincode
	txid, err := assignCurrency(string(assigns), assign.User)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "assign Currency failed."})
		return
	}

	rw.WriteHeader(http.StatusOK)
	encoder.Encode(restResult{OK: txid})
}

// CheckAssign 检测分发币结果
// response说明：StatusBadRequest  失败  不需继续轮询，Error表示失败原因
//				StatusOK OK="1" 成功  不需继续轮询
//				StatusOK OK="0" 未果  需要继续轮询
func (a *AppREST) CheckAssign(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Info("REST processing check assign request...")

	encoder := json.NewEncoder(rw)

	txid := req.PathParams["txid"]
	if txid == "" {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Client must supply a id for checkassign requests."})

		// myLogger.Errorf("Client must supply a id for checkassign requests.")
		return
	}

	v, ok := chaincodeResult[txid]
	if !ok {
		rw.WriteHeader(http.StatusOK)
		encoder.Encode(restResult{OK: "0"})
	} else if v == Chaincode_Success {
		rw.WriteHeader(http.StatusOK)
		encoder.Encode(restResult{OK: "1"})
	} else {
		encoder.Encode(restResult{Err: v})
	}
}

// Exchange 挂单
func (a *AppREST) Exchange(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Info("REST processing order request...")

	encoder := json.NewEncoder(rw)

	// Read in the incoming request payload
	reqBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		encoder.Encode(restResult{Err: "Internal JSON error when reading request body."})

		// myLogger.Error("Internal JSON error when reading request body.")
		return
	}

	// Incoming request body may not be empty, client must supply request payload
	if string(reqBody) == "" {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Client must supply a payload for order requests."})

		// myLogger.Error("Client must supply a payload for order requests.")
		return
	}

	// Payload must conform to the following structure
	var order Order

	// Decode the request payload as an Request structure.	There will be an
	// error here if the incoming JSON is invalid
	err = json.Unmarshal(reqBody, &order)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: fmt.Sprintf("Error unmarshalling order request payload: %s", err)})

		// myLogger.Errorf("Error unmarshalling order request payload: %s", err)
		return
	}

	// 校验请求数据
	if len(order.Account) <= 0 {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Account cann't be empty."})

		// myLogger.Error("Account cann't be empty.")
		return
	}
	if len(order.SrcCurrency) <= 0 {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "SrcCurrency cann't be empty."})

		// myLogger.Error("SrcCurrency cann't be empty.")
		return
	}
	if len(order.DesCurrency) <= 0 {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "DesCurrency cann't be empty."})

		// myLogger.Error("DesCurrency cann't be empty.")
		return
	}
	if order.SrcCount <= 0 {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "SrcCount must be greater than 0."})

		// myLogger.Error("SrcCount must be greater than 0.")
		return
	}
	if order.DesCount <= 0 {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "DesCount must be greater than 0."})

		// myLogger.Error("DesCount must be greater than 0.")
		return
	}

	//将挂单信息保存在待处理队列中
	uuid := util.GenerateUUID()
	order.UUID = uuid
	order.RawUUID = uuid
	order.PendingTime = time.Now().Unix()
	order.PendingDate = time.Now().Format("2006-01-02 15:04:05")

	err = addOrder(uuid, &order)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: fmt.Sprintf("Error redis operation: %s", err)})

		// myLogger.Errorf("Error redis operation: %s", err)
		return
	}

	err = addSet(PendingOrdersKey, uuid)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: fmt.Sprintf("Error redis operation: %s", err)})

		// myLogger.Errorf("Error redis operation: %s", err)
		return
	}

	// myLogger.Debugf("挂单信息: %+v", order)

	rw.WriteHeader(http.StatusOK)
	encoder.Encode(restResult{OK: uuid})
}

// CheckOrder  检测挂单结果，由前端轮询
// response说明：StatusBadRequest  挂单失败  不需继续轮询，Error表示失败原因
//				StatusOK OK="1"   挂单成功  不需继续轮询
//				StatusOK OK="0"   未果 需要继续轮询
func (a *AppREST) CheckOrder(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Info("REST processing check order request...")

	encoder := json.NewEncoder(rw)

	uuid := req.PathParams["uuid"]
	if uuid == "" {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Client must supply a id for checkorder requests."})

		// myLogger.Errorf("Client must supply a id for checkorder requests.")
		return
	}

	// 1.检测该挂单是否在挂单成功队列中
	is, err := isInSet(PendSuccessOrdersKey, uuid)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: fmt.Sprintf("Error redis operation: %s", err)})

		// myLogger.Errorf("Error redis operation: %s", err)
		return
	}
	if is {
		rw.WriteHeader(http.StatusOK)
		encoder.Encode(restResult{OK: "1"})

		// myLogger.Debugf("%s 挂单成功", uuid)

		return
	}

	// 2.检测该挂单是否在挂单失败队列中
	is, err = isInSet(PendFailOrdersKey, uuid)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: fmt.Sprintf("Error redis operation: %s", err)})

		// myLogger.Errorf("Error redis operation: %s", err)
		return
	}
	if is {
		order, err := getOrder(uuid)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			encoder.Encode(restResult{Err: fmt.Sprintf("Error redis operation: %s", err)})

			// myLogger.Errorf("Error redis operation: %s", err)
			return
		}

		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: order.Metadata})

		// myLogger.Debugf("%s 挂单失败", uuid)

		//如果检测到挂单失败，则将该挂单相关信息清除，因为失败的挂单相当于未保存到系统
		go clearFailedOrder(uuid)

		return
	}

	rw.WriteHeader(http.StatusOK)
	encoder.Encode(restResult{OK: "0"})
}

// Cancel 撤单
func (a *AppREST) Cancel(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Info("REST processing cancel order request...")

	encoder := json.NewEncoder(rw)

	// Read in the incoming request payload
	reqBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		encoder.Encode(restResult{Err: "Internal JSON error when reading request body."})

		// myLogger.Error("Internal JSON error when reading request body.")
		return
	}

	uuid := string(reqBody)
	// Incoming request body may not be empty, client must supply request payload
	if uuid == "" {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Client must supply a payload for order requests."})

		// myLogger.Error("Client must supply a payload for order requests.")
		return
	}

	// Payload must conform to the following structure
	// var uuid *struct {
	// 	UUID string `json:"uuid"`
	// }

	// // Decode the request payload as an Request structure.	There will be an
	// // error here if the incoming JSON is invalid
	// err = json.Unmarshal(reqBody, &uuid)
	// if err != nil {
	// 	rw.WriteHeader(http.StatusBadRequest)
	// 	encoder.Encode(restResult{Err: fmt.Sprintf("Error unmarshalling order request payload: %s", err)})

	// 	// myLogger.Errorf("Error unmarshalling order request payload: %s", err)
	// 	return
	// }

	// if len(uuid.UUID) == 0 {
	// 	rw.WriteHeader(http.StatusBadRequest)
	// 	encoder.Encode(restResult{Err: "UUID cann't be empty."})

	// 	// myLogger.Error("UUID cann't be empty.")
	// 	return
	// }

	order, err := getOrder(uuid)

	// 在买卖队列中的（已锁定的）才有撤单
	key := getBSKey(order.SrcCurrency, order.DesCurrency)
	is := isInZSet(key, order.UUID)
	if is {
		// 1.将挂单从买入队列移到待撤单队列中
		err = mvBS2Cancel(key, order.UUID)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			encoder.Encode(restResult{Err: fmt.Sprintf("Error redis operation: %s", err)})

			// myLogger.Errorf("Error redis operation: %s", err)
			return
		}
	} else {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Can't cancel order"})

		// myLogger.Errorf("Error redis operation: %s", err)
		return
	}

	rw.WriteHeader(http.StatusOK)
	encoder.Encode(restResult{OK: order.UUID})
}

// CheckCancel 检查撤单是否成功
// response说明：StatusBadRequest撤单失败  不需继续轮询，Error表示失败原因
//				StatusOK OK="1" 撤单成功  不需继续轮询
//				StatusOK OK="0" 未果 需要继续轮询
func (a *AppREST) CheckCancel(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Info("REST processing check order cancel request...")

	encoder := json.NewEncoder(rw)

	uuid := req.PathParams["uuid"]
	if uuid == "" {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "Client must supply a id for checkorder requests."})

		// myLogger.Errorf("Client must supply a id for checkorder requests.")
		return
	}

	// 1.检测该挂单是否在撤单成功队列中
	is, err := isInSet(CancelSuccessOrderKey, uuid)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: fmt.Sprintf("Error redis operation: %s", err)})

		// myLogger.Errorf("Error redis operation: %s", err)
		return
	}
	if is {
		rw.WriteHeader(http.StatusOK)
		encoder.Encode(restResult{OK: "1"})

		// myLogger.Debugf("%s 撤单成功", uuid)

		return
	}

	// 2.检测该挂单是否在撤单失败队列中
	is, err = isInSet(CancelFailOrderKey, uuid)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: fmt.Sprintf("Error redis operation: %s", err)})

		// myLogger.Errorf("Error redis operation: %s", err)
		return
	}
	if is {
		order, err := getOrder(uuid)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			encoder.Encode(restResult{Err: fmt.Sprintf("Error redis operation: %s", err)})

			// myLogger.Errorf("Error redis operation: %s", err)
			return
		}

		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: order.Metadata})

		// myLogger.Debugf("%s 撤单失败", uuid)

		return
	}

	rw.WriteHeader(http.StatusOK)
	encoder.Encode(restResult{OK: "0"})
}

// Deposit 充值
func (a *AppREST) Deposit(rw web.ResponseWriter, req *web.Request) {
	return
}

// Withdrawals 提现
func (a *AppREST) Withdrawals(rw web.ResponseWriter, req *web.Request) {
	return
}

// login confirms the account and secret password of the client with the
// CA and stores the enrollment certificate and key in the Devops server.
func (s *AppREST) Login(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Debug("------------- login...")

	encoder := json.NewEncoder(rw)

	// Decode the incoming JSON payload
	reqBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: PARAMERR, Msg: "request parameter is wrong"}})
		// myLogger.Errorf("Failed login: [%s]", err)

		return
	}
	// myLogger.Debugf("login request body :%s", string(reqBody))

	var loginRequest User
	err = json.Unmarshal(reqBody, &loginRequest)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: PARAMERR, Msg: "request parameter is wrong"}})
		// myLogger.Errorf("Failed login: [%s]", err)

		return
	}

	// Check that the enrollId and enrollSecret are not left blank.
	if (loginRequest.EnrollID == "") || (loginRequest.EnrollSecret == "") {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResult{Err: "enrollId and enrollSecret can not be null."})
		// myLogger.Errorf("Failed login: [%s]", errors.New("enrollId and enrollSecret can not be null"))

		return
	}

	_, err = setCryptoClient(loginRequest.EnrollID, loginRequest.EnrollSecret)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: SYSERR, Msg: "username or pwd is wrong"}})
		// myLogger.Errorf("Failed login: [%s]", err)

		return
	}

	http.SetCookie(rw, &http.Cookie{
		Name:   "loginfo",
		Value:  loginRequest.EnrollID,
		Path:   "/",
		MaxAge: 86400,
	})

	rw.WriteHeader(http.StatusOK)
	encoder.Encode(restResp{
		Status: SUCCESS,
		Result: struct {
			UserInfo User `json:"userInfo"`
		}{
			UserInfo: User{EnrollID: loginRequest.EnrollID},
		}})
	// myLogger.Debugf("Login successful for user '%s'.", loginRequest.EnrollID)

	// myLogger.Debug("------------- login Done")

	return
}

func (s *AppREST) Logout(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Debug("------------- logout...")

	encoder := json.NewEncoder(rw)

	// 删除cookie
	http.SetCookie(rw, &http.Cookie{
		Name:   "loginfo",
		Path:   "/",
		MaxAge: -1,
	})

	rw.WriteHeader(http.StatusOK)
	encoder.Encode(restResp{Status: SUCCESS})
	// myLogger.Debug("Logout successful.")

	// myLogger.Debug("------------- logout Done")

	return
}

func checkLogin(req *web.Request) (string, error) {
	cookie, err := req.Cookie("loginfo")
	if err != nil || cookie.Value == "" {
		return "", errors.New("not login")
	}

	return cookie.Value, nil
}

// IsLogin IsLogin
func (s *AppREST) IsLogin(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Debug("------------- islogin...")

	encoder := json.NewEncoder(rw)
	enrollID, err := checkLogin(req)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: NOTLOGIN, Msg: err.Error()}})
		// myLogger.Errorf("IsLogout failed: [%s].", err)
		return
	}

	rw.WriteHeader(http.StatusOK)
	encoder.Encode(restResp{
		Status: SUCCESS,
		Result: struct {
			UserInfo User `json:"userInfo"`
		}{
			UserInfo: User{EnrollID: enrollID},
		}})

	// myLogger.Debugf("IsLogout successful for user '%s'.", enrollID)

	// myLogger.Debug("------------- islogin Done")

	return
}

func (s *AppREST) My(rw web.ResponseWriter, req *web.Request) {
	// myLogger.Debug("------------- my...")

	encoder := json.NewEncoder(rw)
	enrollID, err := checkLogin(req)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: NOTLOGIN, Msg: err.Error()}})
		// myLogger.Errorf("My failed: [%s].", err)
		return
	}

	// 获取个人币
	result, _ := getCurrencysByUser(enrollID)
	// if err != nil {
	// 	rw.WriteHeader(http.StatusBadRequest)
	// 	encoder.Encode(restResp{Status: FAILED, Msg: respErr{Code: SYSERR, Msg: "Get currency failed"}})
	// 	// myLogger.Errorf("Get currency failed")
	// 	return
	// }

	var myCurrency []Currency
	_ = json.Unmarshal([]byte(result), &myCurrency)
	// if err != nil {
	// 	rw.WriteHeader(http.StatusBadRequest)
	// 	encoder.Encode(restResp{Status: FAILED, Msg: respErr{Code: SYSERR, Msg: "Get currency failed"}})
	// 	// myLogger.Errorf("Get currency failed")
	// 	return
	// }

	for k, v := range myCurrency {
		myCurrency[k].Count = v.Count / Multiple
		myCurrency[k].LeftCount = v.LeftCount / Multiple
	}

	// 获取个人资产
	result, _ = getAsset(enrollID)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		encoder.Encode(restResp{Status: FAILED, Result: respErr{Code: SYSERR, Msg: "Get owner asset failed"}})
		// myLogger.Errorf("Get owner asset failed")
		return
	}

	var myAsset []Asset
	_ = json.Unmarshal([]byte(result), &myAsset)
	// if err != nil {
	// 	rw.WriteHeader(http.StatusBadRequest)
	// 	encoder.Encode(restResp{Status: FAILED, Msg: respErr{Code: SYSERR, Msg: "Get owner asset failed"}})
	// 	// myLogger.Errorf("Get owner asset failed")
	// 	return
	// }

	for k, v := range myAsset {
		myAsset[k].Count = v.Count / Multiple
		myAsset[k].LockCount = v.LockCount / Multiple
	}

	rw.WriteHeader(http.StatusOK)
	encoder.Encode(restResp{
		Status: SUCCESS,
		Result: struct {
			Currencys []Currency `json:"currencys"`
			Assets    []Asset    `json:"assets"`
		}{
			Currencys: myCurrency,
			Assets:    myAsset,
		},
	})

	// myLogger.Debugf("My successful for user '%s'.", enrollID)

	// myLogger.Debug("------------- my Done")

	return
}
