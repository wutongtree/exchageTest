package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/crypto/primitives"
	logging "github.com/op/go-logging"
)

type ExchangeChaincode struct {
	stub shim.ChaincodeStubInterface
	args []string
}

type Currency struct {
	ID         string `json:"id"`
	Count      int64  `json:"count"`
	LeftCount  int64  `json:"leftCount"`
	Creator    string `json:"creator"`
	CreateTime int64  `json:"createTime"`
}

type Asset struct {
	Owner     string `json:"owner"`
	Currency  string `json:"currency"`
	Count     int64  `json:"count"`
	LockCount int64  `json:"lockCount"`
}

// 定义批量操作的错误类型
// CheckErr 表示校验类错误，这样应该跳过该成员，继续执行批量里的其他成员
// WorldStateErr 表示修改worldstate类错误，此时应直接结束本次交易，批量操作全部失败
type ErrType string

const (
	TableCurrency           = "Currency"
	TableCurrencyReleaseLog = "CurrencyReleaseLog"
	TableCurrencyAssignLog  = "CurrencyAssignLog"
	TableAssets             = "Assets"
	TableAssetLockLog       = "AssetLockLog"
	TableTxLog              = "TxLog"
	TableTxLog2             = "TxLog2"
	CNY                     = "CNY"
	USD                     = "USD"
	CheckErr                = ErrType("CheckErr")
	WorldStateErr           = ErrType("WdErr")
)

var (
	myLogger  = logging.MustGetLogger("exchange_chaincode")
	ExecedErr = errors.New("execed")
	NoDataErr = errors.New("No row data")
)

// ******************注意*****************
// *******所有币数量相关数据，均由APP四舍五入保留六位小数然后乘10^6************
// *********因为chaincode table不支持float类型，不同语言的float精度处理也不同******
// ******************************************************

// Init method will be called during deployment.
// func (c *ExchangeChaincode) Init(stub shim.ChaincodeStubInterface) ([]byte, error) {
func (c *ExchangeChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	myLogger.Debug("Init Chaincode...")

	// _, args := stub.GetFunctionAndParameters()
	if len(args) != 0 {
		return nil, errors.New("Incorrect number of arguments. Expecting 0")
	}

	c.stub = stub
	c.args = args

	err := c.createTable()
	if err != nil {
		myLogger.Errorf("Init error1:%s", err)
		return nil, err
	}

	err = c.initTable()
	if err != nil {
		myLogger.Errorf("Init error2:%s", err)
		return nil, err
	}

	myLogger.Debug("Done.")

	return nil, nil
}

func (c *ExchangeChaincode) createTable() error {
	// 币种信息
	err := c.stub.CreateTable(TableCurrency, []*shim.ColumnDefinition{
		&shim.ColumnDefinition{Name: "ID", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "Count", Type: shim.ColumnDefinition_INT64, Key: false},
		&shim.ColumnDefinition{Name: "LeftCount", Type: shim.ColumnDefinition_INT64, Key: false},
		&shim.ColumnDefinition{Name: "Creator", Type: shim.ColumnDefinition_STRING, Key: false},
		&shim.ColumnDefinition{Name: "CreateTime", Type: shim.ColumnDefinition_INT64, Key: false},
	})
	if err != nil {
		myLogger.Errorf("createTable error1:%s", err)
		return err //errors.New("Failed creating Currency table.")
	}

	// 币发布log
	err = c.stub.CreateTable(TableCurrencyReleaseLog, []*shim.ColumnDefinition{
		&shim.ColumnDefinition{Name: "Currency", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "Count", Type: shim.ColumnDefinition_INT64, Key: false},
		&shim.ColumnDefinition{Name: "ReleaseTime", Type: shim.ColumnDefinition_INT64, Key: true},
	})
	if err != nil {
		myLogger.Errorf("createTable error2:%s", err)
		return errors.New("Failed creating CurrencyReleaseLog table.")
	}

	// 币分发log
	err = c.stub.CreateTable(TableCurrencyAssignLog, []*shim.ColumnDefinition{
		&shim.ColumnDefinition{Name: "Currency", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "Owner", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "Count", Type: shim.ColumnDefinition_INT64, Key: false},
		&shim.ColumnDefinition{Name: "AssignTime", Type: shim.ColumnDefinition_INT64, Key: true},
	})
	if err != nil {
		myLogger.Errorf("createTable error3:%s", err)
		return errors.New("Failed creating CurrencyAssignLog table.")
	}

	// 账户资产信息
	err = c.stub.CreateTable(TableAssets, []*shim.ColumnDefinition{
		&shim.ColumnDefinition{Name: "Owner", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "Currency", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "Count", Type: shim.ColumnDefinition_INT64, Key: false},
		&shim.ColumnDefinition{Name: "LockCount", Type: shim.ColumnDefinition_INT64, Key: false},
	})
	if err != nil {
		myLogger.Errorf("createTable error4:%s", err)
		return errors.New("Failed creating Assets table.")
	}

	// 账户余额锁定log
	err = c.stub.CreateTable(TableAssetLockLog, []*shim.ColumnDefinition{
		&shim.ColumnDefinition{Name: "Owner", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "Currency", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "Order", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "IsLock", Type: shim.ColumnDefinition_BOOL, Key: true},
		&shim.ColumnDefinition{Name: "LockCount", Type: shim.ColumnDefinition_INT64, Key: false},
		&shim.ColumnDefinition{Name: "LockTime", Type: shim.ColumnDefinition_INT64, Key: false},
	})
	if err != nil {
		myLogger.Errorf("createTable error5:%s", err)
		return errors.New("Failed creating AssetLockLog table.")
	}

	// 交易log
	err = c.stub.CreateTable(TableTxLog, []*shim.ColumnDefinition{
		&shim.ColumnDefinition{Name: "Owner", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "SrcCurrency", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "DesCurrency", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "RawOrder", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "Detail", Type: shim.ColumnDefinition_BYTES, Key: true},
	})
	if err != nil {
		myLogger.Errorf("createTable error6:%s", err)
		return errors.New("Failed creating TxLog table.")
	}

	// 交易log
	err = c.stub.CreateTable(TableTxLog2, []*shim.ColumnDefinition{
		&shim.ColumnDefinition{Name: "UUID", Type: shim.ColumnDefinition_STRING, Key: true},
		&shim.ColumnDefinition{Name: "Detail", Type: shim.ColumnDefinition_BYTES, Key: false},
	})
	if err != nil {
		myLogger.Errorf("createTable error7:%s", err)
		return errors.New("Failed creating TxLo2s table.")
	}

	return nil
}

func (c *ExchangeChaincode) initTable() error {
	// 内置人民币CNY和美元USD
	ok, err := c.stub.InsertRow(TableCurrency, shim.Row{Columns: []*shim.Column{
		&shim.Column{Value: &shim.Column_String_{String_: CNY}},
		&shim.Column{Value: &shim.Column_Int64{Int64: 0}},
		&shim.Column{Value: &shim.Column_Int64{Int64: 0}},
		&shim.Column{Value: &shim.Column_String_{String_: "system"}},
		&shim.Column{Value: &shim.Column_Int64{Int64: time.Now().Unix()}},
	}})
	if !ok && err == nil {
		return fmt.Errorf("Failed initiliazing Currency CNY.")
	}
	if err != nil {
		myLogger.Errorf("initTable error2:%s", err)
		return fmt.Errorf("Failed initiliazing Currency CNY: [%s]", err)
	}

	ok, err = c.stub.InsertRow(TableCurrency, shim.Row{Columns: []*shim.Column{
		&shim.Column{Value: &shim.Column_String_{String_: USD}},
		&shim.Column{Value: &shim.Column_Int64{Int64: 0}},
		&shim.Column{Value: &shim.Column_Int64{Int64: 0}},
		&shim.Column{Value: &shim.Column_String_{String_: "system"}},
		&shim.Column{Value: &shim.Column_Int64{Int64: time.Now().Unix()}},
	}})
	if !ok && err == nil {
		return fmt.Errorf("Failed initiliazing Currency USD.")
	}
	if err != nil {
		myLogger.Errorf("initTable error2:%s", err)
		return fmt.Errorf("Failed initiliazing Currency USD: [%s]", err)
	}

	return nil
}

// Invoke will be called for every transaction.
// func (c *ExchangeChaincode) Invoke(stub shim.ChaincodeStubInterface) ([]byte, error) {
func (c *ExchangeChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	// function, args := stub.GetFunctionAndParameters()

	c.stub = stub
	c.args = args

	if function == "createCurrency" {
		return c.createCurrency()
	} else if function == "releaseCurrency" {
		return c.releaseCurrency()
	} else if function == "assignCurrency" {
		return c.assignCurrency()
	} else if function == "exchange" {
		return c.exchange()
	} else if function == "lock" {
		return c.lock()
	}

	return nil, errors.New("Received unknown function invocation")
}

// createCurrency 创建币
// 参数：代号，数量，创建者
func (c *ExchangeChaincode) createCurrency() ([]byte, error) {
	myLogger.Debug("Create Currency...")

	if len(c.args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 3")
	}

	id := c.args[0]
	count, _ := strconv.ParseInt(c.args[1], 10, 64)
	// creator, err := base64.StdEncoding.DecodeString(c.args[2])
	// if err != nil {
	// 	myLogger.Errorf("createCurrency error1:%s", err)
	// 	return nil, errors.New("Failed decodinf creator")
	// }
	creator := c.args[2]
	timestamp := time.Now().Unix()

	ok, err := c.stub.InsertRow(TableCurrency,
		shim.Row{
			Columns: []*shim.Column{
				&shim.Column{Value: &shim.Column_String_{String_: id}},
				&shim.Column{Value: &shim.Column_Int64{Int64: count}},
				&shim.Column{Value: &shim.Column_Int64{Int64: count}},
				&shim.Column{Value: &shim.Column_String_{String_: creator}},
				&shim.Column{Value: &shim.Column_Int64{Int64: timestamp}},
			},
		})
	if err != nil {
		myLogger.Errorf("createCurrency error2:%s", err)
		return nil, errors.New("Failed inserting row.")
	}
	if !ok {
		return nil, errors.New("Currency was already existed.")
	}

	if count > 0 {
		ok, err = c.stub.InsertRow(TableCurrencyReleaseLog,
			shim.Row{
				Columns: []*shim.Column{
					&shim.Column{Value: &shim.Column_String_{String_: id}},
					&shim.Column{Value: &shim.Column_Int64{Int64: count}},
					&shim.Column{Value: &shim.Column_Int64{Int64: timestamp}},
				},
			})
		if err != nil {
			myLogger.Errorf("createCurrency error3:%s", err)
			return nil, errors.New("Failed inserting row.")
		}
		if !ok {
			return nil, errors.New("Currency was already releassed.")
		}
	}

	myLogger.Debug("Done.")
	return nil, nil
}

// releaseCurrency 发布货币
// 参数：代号，数量
func (c *ExchangeChaincode) releaseCurrency() ([]byte, error) {
	myLogger.Debug("Release Currency...")

	if len(c.args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2")
	}

	id := c.args[0]
	count, _ := strconv.ParseInt(c.args[1], 10, 64)

	if id == CNY || id == USD {
		return nil, errors.New("Currency can't be CNY or USD")
	}

	row, curr, err := c.getCurrencyByID(id)
	if err != nil {
		myLogger.Errorf("releaseCurrency error1:%s", err)
		return nil, fmt.Errorf("Failed retrieving currency [%s]: [%s]", id, err)
	}
	if curr == nil {
		return nil, fmt.Errorf("Can't find currency [%s]", id)
	}

	myLogger.Debugf("Creator of [%s] is [% x]", id, curr.Creator)
	// ok, err := c.isCreator(curr.Creator)
	// if err != nil {
	// 	myLogger.Errorf("releaseCurrency error2:%s", err)
	// 	return nil, errors.New("Failed checking currency creator identity")
	// }
	// if !ok {
	// 	return nil, errors.New("The caller is not the creator of the currency")
	// }

	if count <= 0 {
		return nil, errors.New("The currency release count must be > 0")
	}

	row.Columns[1].Value = &shim.Column_Int64{Int64: curr.Count + count}
	row.Columns[2].Value = &shim.Column_Int64{Int64: curr.LeftCount + count}

	ok, err := c.stub.ReplaceRow(TableCurrency, row)
	if err != nil {
		myLogger.Errorf("releaseCurrency error3:%s", err)
		return nil, fmt.Errorf("Failed replacing row [%s]", err)
	}
	if !ok {
		return nil, errors.New("Failed replacing row.")
	}

	ok, err = c.stub.InsertRow(TableCurrencyReleaseLog,
		shim.Row{
			Columns: []*shim.Column{
				&shim.Column{Value: &shim.Column_String_{String_: id}},
				&shim.Column{Value: &shim.Column_Int64{Int64: count}},
				&shim.Column{Value: &shim.Column_Int64{Int64: time.Now().Unix()}},
			},
		})
	if err != nil {
		myLogger.Errorf("releaseCurrency error4:%s", err)
		return nil, errors.New("Failed inserting row.")
	}
	if !ok {
		return nil, errors.New("Currency was already releassed.")
	}

	myLogger.Debug("Done.")
	return nil, nil
}

// assignCurrency 分发币
// 参数：代号，{数量，接收者}
func (c *ExchangeChaincode) assignCurrency() ([]byte, error) {
	myLogger.Debug("Assign Currency...")

	if len(c.args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	assign := struct {
		Currency string `json:"currency"`
		Assigns  []struct {
			Owner string `json:"owner"`
			Count int64  `json:"count"`
		} `json:"assigns"`
	}{}

	err := json.Unmarshal([]byte(c.args[0]), &assign)
	if err != nil {
		myLogger.Errorf("assignCurrency error1:%s", err)
		return nil, fmt.Errorf("Failed unmarshalling assign data [%s]", err)
	}

	if len(assign.Assigns) == 0 {
		return nil, errors.New("Invalid assign data")
	}

	row, curr, err := c.getCurrencyByID(assign.Currency)
	if err != nil {
		myLogger.Errorf("assignCurrency error2:%s", err)
		return nil, fmt.Errorf("Failed retrieving currency [%s]: [%s]", assign.Currency, err)
	}
	if curr == nil {
		return nil, fmt.Errorf("Can't find currency [%s]", assign.Currency)
	}
	myLogger.Debugf("Creator of [%s] is [% x]", assign.Currency, curr.Creator)

	// ok, err := c.isCreator(curr.Creator)
	// if err != nil {
	// 	myLogger.Errorf("assignCurrency error3:%s", err)
	// 	return nil, errors.New("Failed checking currency creator identity")
	// }
	// if !ok {
	// 	return nil, errors.New("The caller is not the creator of the currency")
	// }

	assignCount := int64(0)
	for _, v := range assign.Assigns {
		assignCount += v.Count
	}
	if assignCount > curr.LeftCount {
		return nil, fmt.Errorf("The left count [%d] of currency [%s] is insufficient", curr.LeftCount, assign.Currency)
	}

	for _, v := range assign.Assigns {
		if v.Count <= 0 {
			continue
		}

		// owner, err := base64.StdEncoding.DecodeString(v.Owner)
		// if err != nil {
		// 	myLogger.Errorf("assignCurrency error4:%s", err)
		// 	return nil, errors.New("Failed decodinf owner")
		// }
		owner := v.Owner
		_, err = c.stub.InsertRow(TableCurrencyAssignLog,
			shim.Row{
				Columns: []*shim.Column{
					&shim.Column{Value: &shim.Column_String_{String_: assign.Currency}},
					&shim.Column{Value: &shim.Column_String_{String_: owner}},
					&shim.Column{Value: &shim.Column_Int64{Int64: v.Count}},
					&shim.Column{Value: &shim.Column_Int64{Int64: time.Now().Unix()}},
				},
			})
		if err != nil {
			myLogger.Errorf("assignCurrency error5:%s", err)
			return nil, errors.New("Failed inserting row.")
		}

		assetRow, asset, err := c.getOwnerOneAsset(owner, assign.Currency)
		if err != nil {
			myLogger.Errorf("assignCurrency error6:%s", err)
			return nil, fmt.Errorf("Failed retrieving asset [%s] of the user: [%s]", assign.Currency, err)
		}
		if len(assetRow.Columns) == 0 {
			_, err = c.stub.InsertRow(TableAssets,
				shim.Row{
					Columns: []*shim.Column{
						&shim.Column{Value: &shim.Column_String_{String_: owner}},
						&shim.Column{Value: &shim.Column_String_{String_: assign.Currency}},
						&shim.Column{Value: &shim.Column_Int64{Int64: v.Count}},
						&shim.Column{Value: &shim.Column_Int64{Int64: 0}},
					},
				})
			if err != nil {
				myLogger.Errorf("assignCurrency error7:%s", err)
				return nil, errors.New("Failed inserting row.")
			}
		} else {
			assetRow.Columns[2].Value = &shim.Column_Int64{Int64: asset.Count + v.Count}
			_, err = c.stub.ReplaceRow(TableAssets, assetRow)
		}
		if err != nil {
			myLogger.Errorf("assignCurrency error8:%s", err)
			return nil, errors.New("Failed updating row.")
		}

		curr.LeftCount -= v.Count
	}

	if curr.LeftCount != row.Columns[2].GetInt64() {
		row.Columns[2].Value = &shim.Column_Int64{Int64: curr.LeftCount}
		_, err = c.stub.ReplaceRow(TableCurrency, row)
		if err != nil {
			myLogger.Errorf("assignCurrency error9:%s", err)
			return nil, errors.New("Failed updating row.")
		}
	}

	myLogger.Debug("Done.")
	return nil, nil
}

type FailInfo struct {
	Id   string `json:"id"`
	Info string `json:"info"`
}

type BatchResult struct {
	EventName string     `json:"eventName"`
	SrcMethod string     `json:"srcMethod"`
	Success   []string   `json:""success`
	Fail      []FailInfo `json:"fail"`
}

// lockBalance 锁定货币
// 参数：用户，代号，数量，挂单
func (c *ExchangeChaincode) lock() ([]byte, error) {
	myLogger.Debug("Lock Currency...")

	if len(c.args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 3")
	}

	var lockInfos []struct {
		Owner    string `json:"owner"`
		Currency string `json:"currency"`
		OrderId  string `json:"orderId"`
		Count    int64  `json:"count"`
	}

	err := json.Unmarshal([]byte(c.args[0]), &lockInfos)
	if err != nil {
		myLogger.Errorf("lock error1:%s", err)
		return nil, err
	}
	islock, _ := strconv.ParseBool(c.args[1])

	var successInfos []string
	var failInfos []FailInfo

	for _, v := range lockInfos {
		// owner, err := base64.StdEncoding.DecodeString(v.Owner)
		// if err != nil {
		// 	myLogger.Errorf("lock error2:%s", err)
		// 	failInfos = append(failInfos, FailInfo{Id: v.OrderId, Info: "Failed decodinf owner"})
		// 	continue
		// }
		owner := v.Owner
		// TODO 如果是挂单锁定，要判断目标币存不存在

		err, errType := c.lockOrUnlockBalance(owner, v.Currency, v.OrderId, v.Count, islock)
		if errType == CheckErr {
			failInfos = append(failInfos, FailInfo{Id: v.OrderId, Info: err.Error()})
			continue
		} else if errType == WorldStateErr {
			myLogger.Errorf("lock error3:%s", err)
			return nil, err
		}
		successInfos = append(successInfos, v.OrderId)
	}

	batch := BatchResult{EventName: "chaincode_lock", Success: successInfos, Fail: failInfos, SrcMethod: c.args[2]}
	result, err := json.Marshal(&batch)
	if err != nil {
		myLogger.Errorf("lock error4:%s", err)
		return nil, err
	}
	c.stub.SetEvent(batch.EventName, result)

	myLogger.Debug("Done.")
	return nil, nil
}

type Order struct {
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
}

// exchange 交易
// 参数：挂单1，挂单2
func (c *ExchangeChaincode) exchange() ([]byte, error) {
	myLogger.Debug("Exchange...")

	if len(c.args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	var exchangeOrders []struct {
		BuyOrder  Order `json:"buyOrder"`
		SellOrder Order `json:"sellOrder"`
	}
	err := json.Unmarshal([]byte(c.args[0]), &exchangeOrders)
	if err != nil {
		myLogger.Errorf("exchange error1:%s", err)
		return nil, errors.New("Failed unmarshalling order")
	}

	var successInfos []string
	var failInfos []FailInfo

	for _, v := range exchangeOrders {
		buyOrder := v.BuyOrder
		sellOrder := v.SellOrder
		matchOrder := buyOrder.UUID + "," + sellOrder.UUID

		// TODO 账户证书的处理需要完善
		// buyOwner, _ := base64.StdEncoding.DecodeString(buyOrder.Account)
		// sellOwner, _ := base64.StdEncoding.DecodeString(sellOrder.Account)
		// buyOrder.Account = string(buyOwner)
		// sellOrder.Account = string(sellOwner)

		if buyOrder.SrcCurrency != sellOrder.DesCurrency ||
			buyOrder.DesCurrency != sellOrder.SrcCurrency {
			return nil, errors.New("The exchange is invalid")
		}

		// check 是否交易过
		buyRow, _, err := c.getTxLogByID(buyOrder.UUID)
		if err != nil || len(buyRow.Columns) > 0 {
			myLogger.Errorf("exchange error2:%s", err)
			failInfos = append(failInfos, FailInfo{Id: matchOrder, Info: err.Error()})
			continue
		}
		sellRow, _, err := c.getTxLogByID(sellOrder.UUID)
		if err != nil || len(sellRow.Columns) > 0 {
			myLogger.Errorf("exchange error3:%s", err)
			failInfos = append(failInfos, FailInfo{Id: matchOrder, Info: err.Error()})
			continue
		}
		// execTx
		err, errType := c.execTx(&buyOrder, &sellOrder)
		if errType == CheckErr {
			failInfos = append(failInfos, FailInfo{Id: matchOrder, Info: err.Error()})
			continue
		} else if errType == WorldStateErr {
			myLogger.Errorf("exchange error4:%s", err)
			return nil, err
		}

		// txlog
		err = c.saveTxLog(&buyOrder, &sellOrder)
		if err != nil {
			myLogger.Errorf("exchange error5:%s", err)
			return nil, err
		}

		successInfos = append(successInfos, matchOrder)
	}

	batch := BatchResult{EventName: "chaincode_exchange", Success: successInfos, Fail: failInfos}
	result, err := json.Marshal(&batch)
	if err != nil {
		myLogger.Errorf("exchange error6:%s", err)
		return nil, err
	}
	c.stub.SetEvent(batch.EventName, result)

	myLogger.Debug("Done.")

	return nil, nil
}

func (c *ExchangeChaincode) execTx(buyOrder, sellOrder *Order) (error, ErrType) {
	// 买完为止的挂单结算结余数量
	// 挂单UUID等于原始ID时表示该单交易完成
	if buyOrder.IsBuyAll && buyOrder.UUID == buyOrder.RawUUID {
		unlock, err := c.computeBalance(buyOrder.Account, buyOrder.SrcCurrency, buyOrder.DesCurrency, buyOrder.RawUUID, buyOrder.FinalCost)
		if err != nil {
			myLogger.Errorf("execTx error1:%s", err)
			return errors.New("Failed compute balance"), CheckErr
		}
		myLogger.Debugf("Order %s balance %d", buyOrder.UUID, unlock)
		if unlock > 0 {
			err, errType := c.lockOrUnlockBalance(buyOrder.Account, buyOrder.SrcCurrency, buyOrder.RawUUID, unlock, false)
			if err != nil {
				myLogger.Errorf("execTx error2:%s", err)
				return errors.New("Failed unlock balance"), errType
			}
		}
	}

	// 买单源币锁定数量减少
	buySrcRow, buySrcAsset, err := c.getOwnerOneAsset(buyOrder.Account, buyOrder.SrcCurrency)
	if err != nil {
		myLogger.Errorf("execTx error3:%s", err)
		return fmt.Errorf("Failed retrieving asset [%s] of the user: [%s]", buyOrder.SrcCurrency, err), CheckErr
	}
	if len(buySrcRow.Columns) == 0 {
		return fmt.Errorf("The user have not currency [%s]", buyOrder.SrcCurrency), CheckErr
	}
	buySrcRow.Columns[3].Value = &shim.Column_Int64{Int64: buySrcAsset.LockCount - buyOrder.FinalCost}
	_, err = c.stub.ReplaceRow(TableAssets, buySrcRow)
	if err != nil {
		myLogger.Errorf("execTx error4:%s", err)
		return errors.New("Failed updating row"), WorldStateErr
	}

	// 买单目标币数量增加
	buyDesRow, buyDesAsset, err := c.getOwnerOneAsset(buyOrder.Account, buyOrder.DesCurrency)
	if err != nil {
		myLogger.Errorf("execTx error5:%s", err)
		return fmt.Errorf("Failed retrieving asset [%s] of the user: [%s]", buyOrder.DesCurrency, err), CheckErr
	}
	if len(buyDesRow.Columns) == 0 {
		_, err := c.stub.InsertRow(TableAssets,
			shim.Row{
				Columns: []*shim.Column{
					&shim.Column{Value: &shim.Column_String_{String_: buyOrder.Account}},
					&shim.Column{Value: &shim.Column_String_{String_: buyOrder.DesCurrency}},
					&shim.Column{Value: &shim.Column_Int64{Int64: buyOrder.DesCount}},
					&shim.Column{Value: &shim.Column_Int64{Int64: int64(0)}},
				},
			})
		if err != nil {
			myLogger.Errorf("execTx error6:%s", err)
			return errors.New("Failed inserting row"), WorldStateErr
		}
	} else {
		buyDesRow.Columns[2].Value = &shim.Column_Int64{Int64: buyDesAsset.Count + buyOrder.DesCount}
		_, err = c.stub.ReplaceRow(TableAssets, buyDesRow)
		if err != nil {
			myLogger.Errorf("execTx error7:%s", err)
			return errors.New("Failed updating row"), WorldStateErr
		}
	}

	// 买完为止的挂单结算结余数量
	// 挂单UUID等于原始ID时表示该单交易完成
	if sellOrder.IsBuyAll && sellOrder.UUID == sellOrder.RawUUID {
		unlock, err := c.computeBalance(sellOrder.Account, sellOrder.SrcCurrency, sellOrder.DesCurrency, sellOrder.RawUUID, sellOrder.FinalCost)
		if err != nil {
			myLogger.Errorf("execTx error8:%s", err)
			return errors.New("Failed compute balance"), CheckErr
		}
		myLogger.Debugf("Order %s balance %d", sellOrder.UUID, unlock)
		if unlock > 0 {
			err, errType := c.lockOrUnlockBalance(sellOrder.Account, sellOrder.SrcCurrency, sellOrder.RawUUID, unlock, false)
			if err != nil {
				myLogger.Errorf("execTx error9:%s", err)
				return errors.New("Failed unlock balance"), errType
			}
		}
	}

	// 卖单源币数量减少
	sellSrcRow, sellSrcAsset, err := c.getOwnerOneAsset(sellOrder.Account, sellOrder.SrcCurrency)
	if err != nil {
		myLogger.Errorf("execTx error10:%s", err)
		return fmt.Errorf("Failed retrieving asset [%s] of the user: [%s]", sellOrder.SrcCurrency, err), CheckErr
	}
	if len(sellSrcRow.Columns) == 0 {
		return fmt.Errorf("The user have not currency [%s]", sellOrder.SrcCurrency), CheckErr
	}
	sellSrcRow.Columns[3].Value = &shim.Column_Int64{Int64: sellSrcAsset.LockCount - sellOrder.FinalCost}
	_, err = c.stub.ReplaceRow(TableAssets, sellSrcRow)
	if err != nil {
		myLogger.Errorf("execTx error11:%s", err)
		return errors.New("Failed updating row"), WorldStateErr
	}

	// 卖单目标币数量增加
	sellDesRow, sellDesAsset, err := c.getOwnerOneAsset(sellOrder.Account, sellOrder.DesCurrency)
	if err != nil {
		myLogger.Errorf("execTx error12:%s", err)
		return fmt.Errorf("Failed retrieving asset [%s] of the user: [%s]", sellOrder.DesCurrency, err), CheckErr
	}
	if len(sellDesRow.Columns) == 0 {
		_, err = c.stub.InsertRow(TableAssets,
			shim.Row{
				Columns: []*shim.Column{
					&shim.Column{Value: &shim.Column_String_{String_: sellOrder.Account}},
					&shim.Column{Value: &shim.Column_String_{String_: sellOrder.DesCurrency}},
					&shim.Column{Value: &shim.Column_Int64{Int64: sellOrder.DesCount}},
					&shim.Column{Value: &shim.Column_Int64{Int64: 0}},
				},
			})
		if err != nil {
			myLogger.Errorf("execTx error13:%s", err)
			return errors.New("Failed inserting row"), WorldStateErr
		}
	} else {
		sellDesRow.Columns[2].Value = &shim.Column_Int64{Int64: sellDesAsset.Count + sellOrder.DesCount}
		_, err = c.stub.ReplaceRow(TableAssets, sellDesRow)
		if err != nil {
			myLogger.Errorf("execTx error14:%s", err)
			return errors.New("Failed updating row"), WorldStateErr
		}
	}
	return nil, ErrType("")
}

// saveTxLog 保存交易log
func (c *ExchangeChaincode) saveTxLog(buyOrder, sellOrder *Order) error {
	buyJson, _ := json.Marshal(buyOrder)
	sellJson, _ := json.Marshal(sellOrder)

	_, err := c.stub.InsertRow(TableTxLog, shim.Row{
		Columns: []*shim.Column{
			&shim.Column{Value: &shim.Column_String_{String_: buyOrder.Account}},
			&shim.Column{Value: &shim.Column_String_{String_: buyOrder.SrcCurrency}},
			&shim.Column{Value: &shim.Column_String_{String_: buyOrder.DesCurrency}},
			&shim.Column{Value: &shim.Column_String_{String_: buyOrder.RawUUID}},
			&shim.Column{Value: &shim.Column_Bytes{Bytes: buyJson}},
		},
	})
	if err != nil {
		myLogger.Errorf("saveTxLog error1:%s", err)
		return errors.New("Failed inserting row")
	}

	_, err = c.stub.InsertRow(TableTxLog2, shim.Row{
		Columns: []*shim.Column{
			&shim.Column{Value: &shim.Column_String_{String_: buyOrder.UUID}},
			&shim.Column{Value: &shim.Column_Bytes{Bytes: buyJson}},
		},
	})
	if err != nil {
		myLogger.Errorf("saveTxLog error2:%s", err)
		return errors.New("Failed inserting row")
	}

	_, err = c.stub.InsertRow(TableTxLog, shim.Row{
		Columns: []*shim.Column{
			&shim.Column{Value: &shim.Column_String_{String_: sellOrder.Account}},
			&shim.Column{Value: &shim.Column_String_{String_: sellOrder.SrcCurrency}},
			&shim.Column{Value: &shim.Column_String_{String_: sellOrder.DesCurrency}},
			&shim.Column{Value: &shim.Column_String_{String_: sellOrder.RawUUID}},
			&shim.Column{Value: &shim.Column_Bytes{Bytes: sellJson}},
		},
	})
	if err != nil {
		myLogger.Errorf("saveTxLog error3:%s", err)
		return errors.New("Failed inserting row")
	}

	_, err = c.stub.InsertRow(TableTxLog2, shim.Row{
		Columns: []*shim.Column{
			&shim.Column{Value: &shim.Column_String_{String_: sellOrder.UUID}},
			&shim.Column{Value: &shim.Column_Bytes{Bytes: sellJson}},
		},
	})
	if err != nil {
		myLogger.Errorf("saveTxLog error4:%s", err)
		return errors.New("Failed inserting row")
	}
	return nil
}

func (c *ExchangeChaincode) isCreator(certificate []byte) (bool, error) {
	// In order to enforce access control, we require that the
	// metadata contains the following items:
	// 1. a certificate Cert
	// 2. a signature Sigma under the signing key corresponding
	// to the verification key inside Cert of :
	// (a) Cert;
	// (b) The payload of the transaction (namely, function name and args) and
	// (c) the transaction binding.
	// Verify Sigma=Sign(certificate.sk, Cert||tx.Payload||tx.Binding) against Cert.vk

	sigma, err := c.stub.GetCallerMetadata()
	if err != nil {
		myLogger.Errorf("isCreator error1:%s", err)
		return false, errors.New("Failed getting metadata")
	}

	payload, err := c.stub.GetPayload()
	if err != nil {
		myLogger.Errorf("isCreator error2:%s", err)
		return false, errors.New("Failed getting payload")
	}

	binding, err := c.stub.GetBinding()
	if err != nil {
		myLogger.Errorf("isCreator error3:%s", err)
		return false, errors.New("Failed getting binding")
	}

	myLogger.Debugf("passed certificate [% x]", certificate)
	myLogger.Debugf("passed sigma [% x]", sigma)
	myLogger.Debugf("passed payload [% x]", payload)
	myLogger.Debugf("passed binding [% x]", binding)

	ok, err := c.stub.VerifySignature(certificate, sigma, append(payload, binding...))
	if err != nil {
		myLogger.Errorf("isCreator error4:%s", err)
		myLogger.Errorf("Failed checking signature [%s]", err)
		return ok, err
	}
	if !ok {
		myLogger.Error("Invalid signature")
	}

	myLogger.Debug("Check ...Verified!")

	return true, nil
}

func (c *ExchangeChaincode) getCurrencyByID(id string) (shim.Row, *Currency, error) {
	var currency *Currency

	row, err := c.stub.GetRow(TableCurrency, []shim.Column{
		shim.Column{Value: &shim.Column_String_{String_: id}},
	})

	if len(row.Columns) > 0 {
		currency = &Currency{
			ID:         row.Columns[0].GetString_(),
			Count:      row.Columns[1].GetInt64(),
			LeftCount:  row.Columns[2].GetInt64(),
			Creator:    row.Columns[3].GetString_(),
			CreateTime: row.Columns[4].GetInt64(),
		}
	}
	return row, currency, err
}

func (c *ExchangeChaincode) getAllCurrency() ([]shim.Row, []*Currency, error) {
	rowChannel, err := c.stub.GetRows(TableCurrency, nil)
	if err != nil {
		myLogger.Errorf("getAllCurrency error1:%s", err)
		return nil, nil, fmt.Errorf("getRows operation failed. %s", err)
	}
	var rows []shim.Row
	var infos []*Currency
	for {
		select {
		case row, ok := <-rowChannel:
			if !ok {
				rowChannel = nil
			} else {
				rows = append(rows, row)

				info := new(Currency)
				info.ID = row.Columns[0].GetString_()
				info.Count = row.Columns[1].GetInt64()
				info.LeftCount = row.Columns[2].GetInt64()
				info.Creator = row.Columns[3].GetString_()
				info.CreateTime = row.Columns[4].GetInt64()

				infos = append(infos, info)
			}
		}
		if rowChannel == nil {
			break
		}
	}
	return rows, infos, nil
}

func (c *ExchangeChaincode) getOwnerOneAsset(owner string, currency string) (shim.Row, *Asset, error) {
	var asset *Asset

	row, err := c.stub.GetRow(TableAssets, []shim.Column{
		shim.Column{Value: &shim.Column_String_{String_: owner}},
		shim.Column{Value: &shim.Column_String_{String_: currency}},
	})

	if len(row.Columns) > 0 {
		asset = &Asset{
			Owner:     row.Columns[0].GetString_(),
			Currency:  row.Columns[1].GetString_(),
			Count:     row.Columns[2].GetInt64(),
			LockCount: row.Columns[3].GetInt64(),
		}
	}

	return row, asset, err
}

func (c *ExchangeChaincode) getOwnerAllAsset(owner string) ([]shim.Row, []*Asset, error) {
	rowChannel, err := c.stub.GetRows(TableAssets, []shim.Column{
		shim.Column{Value: &shim.Column_String_{String_: owner}},
	})
	if err != nil {
		myLogger.Errorf("getOwnerAllAsset error1:%s", err)
		return nil, nil, fmt.Errorf("getOwnerAllAsset operation failed. %s", err)
	}

	var rows []shim.Row
	var assets []*Asset
	for {
		select {
		case row, ok := <-rowChannel:
			if !ok {
				rowChannel = nil
			} else {
				rows = append(rows, row)

				asset := &Asset{
					Owner:     row.Columns[0].GetString_(),
					Currency:  row.Columns[1].GetString_(),
					Count:     row.Columns[2].GetInt64(),
					LockCount: row.Columns[3].GetInt64(),
				}
				assets = append(assets, asset)
			}
		}
		if rowChannel == nil {
			break
		}
	}
	return rows, assets, nil
}

func (c *ExchangeChaincode) lockOrUnlockBalance(owner string, currency, order string, count int64, islock bool) (error, ErrType) {

	row, asset, err := c.getOwnerOneAsset(owner, currency)
	if err != nil {
		myLogger.Errorf("lockOrUnlockBalance error1:%s", err)
		return fmt.Errorf("Failed retrieving asset [%s] of the user: [%s]", currency, err), CheckErr
	}
	if len(row.Columns) == 0 {
		return fmt.Errorf("The user have not currency [%s]", currency), CheckErr
	}
	if islock && asset.Count < count {
		return fmt.Errorf("Currency [%s] of the user is insufficient", currency), CheckErr
	} else if !islock && asset.LockCount < count {
		return fmt.Errorf("Locked currency [%s] of the user is insufficient", currency), CheckErr
	}

	// 判断是否锁定过或解锁过，因为是批量操作，可能会有重复数据。其他批量操作也要作此判断
	lockRow, err := c.getLockLog(owner, currency, order, islock)
	if err != nil {
		myLogger.Errorf("lockOrUnlockBalance error2:%s", err)
		return err, CheckErr
	}
	if len(lockRow.Columns) > 0 {
		myLogger.Errorf("lockOrUnlockBalance error21:%s, order:%s, IsLock:%s, count:%d", ExecedErr, order, islock, count)
		return ExecedErr, CheckErr
	}

	if islock {
		row.Columns[2].Value = &shim.Column_Int64{Int64: asset.Count - count}
		row.Columns[3].Value = &shim.Column_Int64{Int64: asset.LockCount + count}
	} else {
		row.Columns[2].Value = &shim.Column_Int64{Int64: asset.Count + count}
		row.Columns[3].Value = &shim.Column_Int64{Int64: asset.LockCount - count}
	}

	_, err = c.stub.ReplaceRow(TableAssets, row)
	if err != nil {
		myLogger.Errorf("lockOrUnlockBalance error3:%s", err)
		return errors.New("Failed updating row."), WorldStateErr
	}

	_, err = c.stub.InsertRow(TableAssetLockLog,
		shim.Row{
			Columns: []*shim.Column{
				&shim.Column{Value: &shim.Column_String_{String_: owner}},
				&shim.Column{Value: &shim.Column_String_{String_: currency}},
				&shim.Column{Value: &shim.Column_String_{String_: order}},
				&shim.Column{Value: &shim.Column_Bool{Bool: islock}},
				&shim.Column{Value: &shim.Column_Int64{Int64: count}},
				&shim.Column{Value: &shim.Column_Int64{Int64: time.Now().Unix()}},
			},
		})
	if err != nil {
		myLogger.Errorf("lockOrUnlockBalance error4:%s", err)
		return errors.New("Failed inserting row."), WorldStateErr
	}

	return nil, ErrType("")
}

// computeBalance 计算挂单结余
func (c *ExchangeChaincode) computeBalance(owner string, srcCurrency, desCurrency, rawUUID string, currentCost int64) (int64, error) {
	_, txs, err := c.getTXs(owner, srcCurrency, desCurrency, rawUUID)
	if err != nil {
		myLogger.Errorf("computeBalance error1:%s", err)
		return 0, err
	}
	row, err := c.getLockLog(owner, srcCurrency, rawUUID, true)
	if err != nil {
		myLogger.Errorf("computeBalance error2:%s", err)
		return 0, err
	}
	if len(row.Columns) == 0 {
		return 0, errors.New("can't find lock log")
	}

	lock := row.Columns[4].GetInt64()
	sumCost := int64(0)
	for _, tx := range txs {
		sumCost += tx.FinalCost
	}

	return lock - sumCost - currentCost, nil
}

func (c *ExchangeChaincode) getTXs(owner string, srcCurrency, desCurrency, rawOrder string) ([]shim.Row, []*Order, error) {
	rowChannel, err := c.stub.GetRows(TableTxLog, []shim.Column{
		shim.Column{Value: &shim.Column_String_{String_: owner}},
		shim.Column{Value: &shim.Column_String_{String_: srcCurrency}},
		shim.Column{Value: &shim.Column_String_{String_: desCurrency}},
		shim.Column{Value: &shim.Column_String_{String_: rawOrder}},
	})
	if err != nil {
		myLogger.Errorf("getTXs error1:%s", err)
		return nil, nil, fmt.Errorf("getTXs operation failed. %s", err)
	}

	var rows []shim.Row
	var orders []*Order
	for {
		select {
		case row, ok := <-rowChannel:
			if !ok {
				rowChannel = nil
			} else {
				rows = append(rows, row)

				order := new(Order)
				err := json.Unmarshal(row.Columns[4].GetBytes(), order)
				if err != nil {
					myLogger.Errorf("getTXs error2:%s", err)
					return nil, nil, fmt.Errorf("Error unmarshaling JSON: %s", err)
				}

				orders = append(orders, order)
			}
		}
		if rowChannel == nil {
			break
		}
	}
	return rows, orders, nil
}

func (c *ExchangeChaincode) getTxLogByID(uuid string) (shim.Row, *Order, error) {
	var order *Order
	row, err := c.stub.GetRow(TableTxLog2, []shim.Column{
		shim.Column{Value: &shim.Column_String_{String_: uuid}},
	})
	if len(row.Columns) > 0 {
		err = json.Unmarshal(row.Columns[1].GetBytes(), order)
	}

	return row, order, err
}

func (c *ExchangeChaincode) getLockLog(owner string, currency, order string, islock bool) (shim.Row, error) {
	return c.stub.GetRow(TableAssetLockLog, []shim.Column{
		shim.Column{Value: &shim.Column_String_{String_: owner}},
		shim.Column{Value: &shim.Column_String_{String_: currency}},
		shim.Column{Value: &shim.Column_String_{String_: order}},
		shim.Column{Value: &shim.Column_Bool{Bool: islock}},
	})
}

// Query callback representing the query of a chaincode
// Anyone can invoke this function.
func (c *ExchangeChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	myLogger.Debug("Query Chaincode....")

	// function, args, _ = dealParam(function, args)
	c.stub = stub
	c.args = args

	if function == "queryCurrencyByID" {
		return c.queryCurrencyByID()
	} else if function == "queryAllCurrency" {
		return c.queryAllCurrency()
	} else if function == "queryTxLogs" {
		return c.queryTxLogs()
	} else if function == "queryAssetByOwner" {
		return c.queryAssetByOwner()
	} else if function == "queryMyCurrency" {
		return c.queryMyCurrency()
	}

	return nil, errors.New("Received unknown function query")
}

// queryMyCurrency 查询个人创建的币
func (c *ExchangeChaincode) queryMyCurrency() ([]byte, error) {
	myLogger.Debug("queryCurrency...")

	if len(c.args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	owner := c.args[0]
	_, infos, err := c.getAllCurrency()
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, NoDataErr
	}

	var currencys []*Currency
	for _, v := range infos {
		if owner == v.Creator || v.ID == USD || v.ID == CNY {
			currencys = append(currencys, v)
		}
	}
	return json.Marshal(&currencys)
}

// queryAssetByOwner 查询个人资产
func (c *ExchangeChaincode) queryAssetByOwner() ([]byte, error) {
	myLogger.Debug("queryAssetByOwner...")

	if len(c.args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	owner := c.args[0]
	_, assets, err := c.getOwnerAllAsset(owner)
	if err != nil {
		myLogger.Errorf("queryAssetByOwner error1:%s", err)
		return nil, err
	}
	if len(assets) == 0 {
		return nil, NoDataErr
	}
	return json.Marshal(&assets)
}

// queryCurrency 查询币
func (c *ExchangeChaincode) queryCurrencyByID() ([]byte, error) {
	myLogger.Debug("queryCurrency...")

	if len(c.args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	id := c.args[0]

	_, currency, err := c.getCurrencyByID(id)
	if err != nil {
		myLogger.Errorf("queryCurrencyByID error1:%s", err)
		return nil, err
	}
	if currency == nil {
		return nil, NoDataErr
	}

	return json.Marshal(&currency)
}

// queryAllCurrency
func (c *ExchangeChaincode) queryAllCurrency() ([]byte, error) {
	myLogger.Debug("queryCurrency...")

	if len(c.args) != 0 {
		return nil, errors.New("Incorrect number of arguments. Expecting 0")
	}

	_, infos, err := c.getAllCurrency()
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, NoDataErr
	}

	return json.Marshal(&infos)
}

// queryTxLogs
func (c *ExchangeChaincode) queryTxLogs() ([]byte, error) {
	myLogger.Debug("queryTxLogs...")

	if len(c.args) != 0 {
		return nil, errors.New("Incorrect number of arguments. Expecting 0")
	}

	rowChannel, err := c.stub.GetRows(TableTxLog2, nil)
	if err != nil {
		myLogger.Errorf("queryTxLogs error1:%s", err)
		return nil, fmt.Errorf("getRows operation failed. %s", err)
	}

	var infos []*Order
	for {
		select {
		case row, ok := <-rowChannel:
			if !ok {
				rowChannel = nil
			} else {
				info := new(Order)
				err = json.Unmarshal(row.Columns[1].GetBytes(), info)
				if err == nil {
					myLogger.Errorf("queryTxLogs error2:%s", err)
					infos = append(infos, info)
				}
			}
		}
		if rowChannel == nil {
			break
		}
	}

	return json.Marshal(&infos)
}

func main() {
	primitives.SetSecurityLevel("SHA3", 256)
	err := shim.Start(new(ExchangeChaincode))
	if err != nil {
		myLogger.Errorf("mian error1:%s", err)
		fmt.Printf("Error starting exchange chaincode: %s", err)
	}
}
