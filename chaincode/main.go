package main

import (
	"log"
	"os"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// CCAAS 模式下，链码同时具备两种能力：
//  1) 作为 gRPC 服务监听 CHAINCODE_SERVER_ADDRESS，供 peer 反连
//  2) 通过 -peer.address 主动向 peer 注册（shim.Start）
// 两者由 peer 的 connection.json 决定，这里按环境变量选择。
func main() {
	chaincode, err := contractapi.NewChaincode(&XLabContract{})
	if err != nil {
		log.Panicf("create xlab chaincode: %v", err)
	}

	if addr := os.Getenv("CHAINCODE_SERVER_ADDRESS"); addr != "" {
		server := &shim.ChaincodeServer{
			CCID:     os.Getenv("CORE_CHAINCODE_ID_NAME"),
			Address:  addr,
			CC:       chaincode,
			TLSProps: shim.TLSProperties{Disabled: true},
		}
		log.Printf("xlab chaincode server listening on %s (ccid=%s)", addr, server.CCID)
		if err := server.Start(); err != nil {
			log.Panicf("start chaincode server: %v", err)
		}
		return
	}

	if err := chaincode.Start(); err != nil {
		log.Panicf("start xlab chaincode: %v", err)
	}
}
