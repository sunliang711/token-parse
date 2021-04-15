package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"math/big"
	"net/http"
	"sync"
	"time"
	"token-parse/config"
	"token-parse/db"
	"token-parse/tools"
)

const (
	methodGetLogs      = "eth_getLogs"
	transferEventTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
)

func init() {

}

// RPCReq json rpc request struct
type RPCReq struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	ID      uint          `json:"id"`
	Params  []interface{} `json:"params"`
}

type LogParam struct {
	Topics    []string
	FromBlock string `json:"fromBlock,omitempty"`
	ToBlock   string `json:"toBlock,omitempty"`
	Address   string `json:"address,omitempty"`
	BlockHash string `json:"blockHash,omitempty"`
}

// jsonrpc log response
type RPCLogRes struct {
	JSONRPC string `json:"jsonrpc"`
	ID      uint
	Result  []LogRes
}

type LogRes struct {
	Address          string   `json:"address"`
	BlockHash        string   `json:"blockHash"`
	BlockNumber      string   `json:"blockNumber"`
	Data             string   `json:"data"`
	LogIndex         string   `json:"logIndex"`
	Removed          bool     `json:"removed"`
	Topics           []string `json:"topics"`
	TransactionHash  string   `json:"transactionHash"`
	TransactionIndex string   `json:"transactionIndex"`
}

// Log response format:
// {
//  "address":"0xdac17f958d2ee523a2206206994597c13d831ec7",
//  "blockHash":"0xe21e27206bb9b2ca217f478733ea93defeca4ee029e6decbb89de289c83325af",
//  "blockNumber":"0xb00fd8",
//  "data":"0x000000000000000000000000000000000000000000000000000000000876a428",
//  "logIndex":"0x9",
//  "removed":false,
//  "topics":[
//             "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
//             "0x0000000000000000000000005041ed759dd4afc3a72b8192c143f72f4724081a",
//             "0x000000000000000000000000d6ea6a790fce0fa7a07d435d80aab8cf8f87b903"
//            ],
//  "transactionHash":"0xa43be2d1e7a546d7abf58ae5c01e5cb4165cad5865a5aa04a0f0f1b88953cb21",
//  "transactionIndex":"0x4"
// }

// RPC request format:
// curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getLogs","params":[{"topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"],"fromBlock":"0xB00FD8","address":"0x6e1A19F235bE7ED8E3369eF73b196C07257494DE"}],"id":74}'

type Parser struct {
	Config      *config.TokenConfig
	ChQuit      chan struct{}
	Wg          *sync.WaitGroup
	DBOp        *gorm.DB
	HTTPClient  *http.Client
	ChRPCLogRes chan *RPCLogRes
}

const (
	chanSize = 20
)

func New(cfg *config.TokenConfig, dbOp *gorm.DB, wg *sync.WaitGroup) *Parser {
	return &Parser{
		Config: cfg,
		ChQuit: make(chan struct{}),
		Wg:     wg,
		DBOp:   dbOp,
		HTTPClient: &http.Client{
			Timeout: time.Second * time.Duration(cfg.Timeout),
		},
		ChRPCLogRes: make(chan *RPCLogRes, chanSize),
	}
}

// start loop
func (parser *Parser) Start() {
	workTicker := time.Tick(time.Second * time.Duration(parser.Config.Interval))
	fromBlock, err := parser.fromBlock()
	if err != nil {
		fmt.Printf("error: %v", err)
		parser.Stop()
		return
	}
	fmt.Printf("%s From block: %v\n", parser.Config.TokenName, fromBlock)
	go parser.receiveLog()

	for {
		select {
		case <-parser.ChQuit:
			return
		case <-workTicker:
			parser.fetchBlock(fromBlock, parser.Config.BlockStep)
			fromBlock += parser.Config.BlockStep
		}
	}
}

func (parser *Parser) Stop() {
	close(parser.ChQuit)
}

func (parser *Parser) receiveLog() {
	for {
		select {
		case <-parser.ChQuit:
			parser.stopAll()
			return
		case logRes := <-parser.ChRPCLogRes:
			parser.toDB(logRes)
		}
	}
}

func (parser *Parser) toDB(logRes *RPCLogRes) {
	fmt.Printf("write to DB:%+v \n", logRes.Result)

	for i := range logRes.Result {
		result := logRes.Result[i]

		if len(logRes.Result[i].Topics) != 3 || logRes.Result[i].Topics[0] != transferEventTopic {
			continue
		}

		if result.Topics[1] == result.Topics[2] {
			continue
		}

		blockHash := result.BlockHash
		blockNumber, err := tools.HexString2Uint(result.BlockNumber)
		if err != nil {
			fmt.Printf("convert block number error: %v", err)
			return
		}
		txHash := result.TransactionHash
		txIndex, err := tools.HexString2Uint(result.TransactionIndex)
		if err != nil {
			fmt.Printf("convert tx index error: %v", err)
			return
		}
		logIndex, err := tools.HexString2Uint(result.LogIndex)
		if err != nil {
			fmt.Printf("convert log index error: %v", err)
			return
		}
		contractAddress := result.Address

		transfer := big.NewInt(0)
		// base 用0，会自动识别
		transfer.SetString(result.Data, 0)

		fromAddress := result.Topics[1]
		toAddress := result.Topics[2]

		// 读取当前余额，没有默认使用0
		fromBalance, err := parser.getLatestBalance(fromAddress)
		if err != nil {
			logrus.Errorf("get sender balance error: %v", err)
			return
		}

		// 读取当前余额，没有默认使用0
		toBalance, err := parser.getLatestBalance(toAddress)
		if err != nil {
			logrus.Errorf("get receiver balance error: %v", err)
			return
		}

		fromBalance.Sub(fromBalance, transfer)
		toBalance.Add(toBalance, transfer)

		newFromBalanceDAO := &db.BalanceDAO{
			TokenName:        parser.Config.TokenName,
			BlockHash:        blockHash,
			BlockNumber:      uint(blockNumber),
			LogIndex:         uint(logIndex),
			ContractAddress:  contractAddress,
			TransactionHash:  txHash,
			TransactionIndex: uint(txIndex),
			Address:          fromAddress,
			Balance:          fromBalance.Text(10),
		}

		newToBalanceDAO := &db.BalanceDAO{
			TokenName:        parser.Config.TokenName,
			BlockHash:        blockHash,
			BlockNumber:      uint(blockNumber),
			LogIndex:         uint(logIndex),
			ContractAddress:  contractAddress,
			TransactionHash:  txHash,
			TransactionIndex: uint(txIndex),
			Address:          toAddress,
			Balance:          toBalance.Text(10),
		}

		createResult := parser.DBOp.WithContext(tools.GetContextDefault()).Create(newFromBalanceDAO)
		err = createResult.Error
		if err != nil {
			fmt.Printf("insert new from balance error: %v", err)
		}
		createResult = parser.DBOp.WithContext(tools.GetContextDefault()).Create(newToBalanceDAO)
		err = createResult.Error
		if err != nil {
			fmt.Printf("insert new from balance error: %v", err)
		}
	}

}

func (parser *Parser) getLatestBalance(address string) (*big.Int, error) {
	var balance db.BalanceDAO
	addressBalance := big.NewInt(0)
	// select * from db_balance  where address="addr2" order by block_number desc,log_index desc limit 1;
	result := parser.DBOp.WithContext(tools.GetContextDefault()).Model(&db.BalanceDAO{}).Where("token_name = ? and address = ? ", parser.Config.TokenName, address).Order("block_number desc,log_index desc").First(&balance)
	err := result.Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
		} else {
			// db error
			return nil, fmt.Errorf("query latest balance error: %v", err)
		}
	} else {
		_, ok := addressBalance.SetString(balance.Balance, 10)
		if !ok {
			return nil, fmt.Errorf("set balance error")
		}
	}
	return addressBalance, nil
}

func (parser *Parser) fromBlock() (uint, error) {
	// 如果数据库中有数据（不是第一次），那么把最大的block_number所有记录删除，
	// 然后从这个block_number开始

	var balance db.BalanceDAO
	// get max block_number
	result := parser.DBOp.WithContext(tools.GetContextDefault()).Model(&db.BalanceDAO{}).Where("token_name = ? ", parser.Config.TokenName).Order("block_number desc").First(&balance)
	err := result.Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// use parser.Config.FromBlock for the first time
			return parser.Config.FromBlock, nil
		}
		fmt.Printf("get latest block number error: %v", result.Error)
		return 0, err
	}

	// delete all record with block_number == balance.BlockNumber
	result = parser.DBOp.WithContext(tools.GetContextDefault()).Where("token_name = ? and block_number = ? ", parser.Config.TokenName, balance.BlockNumber).Delete(&db.BalanceDAO{})
	err = result.Error
	if err != nil {
		return 0, fmt.Errorf("delete max block_number records error: %v", err)
	}
	fmt.Printf("delete block_number: %v success,check db\n", balance.BlockNumber)
	return balance.BlockNumber, nil
}

func (parser *Parser) fetchBlock(fromBlock uint, blockStep uint) error {
	fmt.Printf("%s fetch block: %v step: %v\n", parser.Config.TokenName, fromBlock, blockStep)
	// 组装eth_getLogs参数
	logParam := LogParam{
		Topics:    []string{transferEventTopic},
		FromBlock: fmt.Sprintf("%#x", fromBlock),
		ToBlock:   fmt.Sprintf("%#x", fromBlock+blockStep-1),
		Address:   parser.Config.ContractAddress,
	}
	rpcID := uint(1)
	// 构建完整的json rpc参数
	rpcReq := RPCReq{
		JSONRPC: "2.0",
		Method:  methodGetLogs,
		ID:      rpcID,
		Params:  []interface{}{&logParam},
	}
	bs, err := json.Marshal(&rpcReq)
	if err != nil {
		return fmt.Errorf("marshal request body error: %v", err)
	}
	fmt.Printf("Request body: %s\n", bs)

	// 构建HTTP请求
	req, err := http.NewRequest("POST", parser.Config.RPC, bytes.NewBuffer(bs))
	if err != nil {
		return fmt.Errorf("new http request error: %v", err)
	}
	req.Header.Add("Content-Type", "application/json")

	fmt.Printf("request URI: %v\n", req.URL)

	// 发送rpc调用
	resp, err := parser.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("call rpc error: %v", err)
	}
	defer resp.Body.Close()

	// 解析结果
	rpcRes := &RPCLogRes{}
	err = json.NewDecoder(resp.Body).Decode(rpcRes)
	if err != nil {
		return fmt.Errorf("decode response error: %v", err)
	}
	if rpcRes.ID != rpcID {
		return fmt.Errorf("jsonrpc id not match")
	}
	fmt.Printf("rpcRes: %+v\n", rpcRes)

	parser.ChRPCLogRes <- rpcRes
	return nil
}

func (parser *Parser) stopAll() {
	if parser.Wg != nil {
		parser.Wg.Done()
		logrus.Info("done")
	}
}
