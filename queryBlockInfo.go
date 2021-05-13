package main


import (
	"github.com/aiguo186/fabric-sdk-go-gm/pkg/core/config"
	"github.com/aiguo186/fabric-sdk-go-gm/pkg/fabsdk"
	"log"
	"github.com/aiguo186/fabric-sdk-go-gm/pkg/client/ledger"


)

const (
	ChannelID = "mychannel"
)



func main() {
	sdk, err := fabsdk.New(config.FromFile("./config.yaml"))
	if err != nil {
		log.Panicf("failed to create fabric sdk: %s", err)
	}

	ccp := sdk.ChannelContext(ChannelID, fabsdk.WithUser("Admin"))

	client, err := ledger.New(ccp)
        if err != nil {
	    log.Panicf("failed to new client: %s", err)
	}

        blockchain, err := client.QueryInfo()
        if err != nil {
            log.Panicf("failed to quneryinfo: %s", err)
        }
        log.Printf("blockinfo: %v",blockchain)

	block, err := client.QueryBlock(2)
        if err != nil {
            log.Panicf("failed to queryblock: %s", err)
        }
        log.Printf("block: %v",block)

}



