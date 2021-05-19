package main


import (
	"github.com/aiguo186/fabric-sdk-go-gm/pkg/core/config"
	"github.com/aiguo186/fabric-sdk-go-gm/pkg/fabsdk"
	"github.com/aiguo186/fabric-sdk-go-gm/third_party/github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/common"
	"log"
	"github.com/aiguo186/fabric-sdk-go-gm/pkg/client/ledger"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-protos-go/peer"
	//"github.com/tidwall/gjson"
)

const (
	ChannelID = "mychannel"
)
type txInfo struct {
	TxID      string
	Creator   string
	Timestamp int64
	Key       string
	Content   []content
}

type value struct {
	ID                string `json:"id"`
	SerialNumber      string `json:"serial_number"`
	OriginFileHash    string `json:"origin_file_hash"`
	AttestationHash   string `json:"attestation_hash"`
	Status            string `json:"status"`
	CreatedAt         string `json:"created_at"`
	DataJSON          string `json:"data_json"`
	AttestationTypeID string `json:"attestation_type_id"`
	TemplateID        string `json:"template_id"`
	OrganizationID    string `json:"organization_id"`
	OrganizationName  string `json:"organization_name"`
	BusinessType      string `json:"business_type"`
	CollectionType    string `json:"collection_type"`
	Tableinfo         string `json:"tableinfo"`
}

type content struct {
	HashAlgo string `json:"hash_algo"`
	Hash     string `json:"hash"`
	URI      string `json:"uri"`
	Meta     string `json:"meta"`
}



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
	log.Printf("blockinfo: %v", blockchain)

	block, err := client.QueryBlock(7)
	if err != nil {
		log.Panicf("failed to queryblock: %s", err)
	}
	//log.Printf("block: %v", block)

	err = GetTxInfosFromEnvelope(block.Data.Data[0])
	if err != nil {
		log.Panicf("failed parse block info: %s", err)
	}
}


func  GetTxInfosFromEnvelope(envBytes []byte) (error) {
	env, err := GetEnvelopeFromBlock(envBytes)
	if err != nil {
		return err
	}
	payload, err := UnmarshalPayload(env.Payload)
	if err != nil {
		return err
	}

	//忽略payload为空的chaincode调用
	if payload == nil {
		log.Println("ignore empty payload chaincode invocation")
		return nil
	}

	channelHeader, err := UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return err
	}

	log.Println("channel id is:",channelHeader.ChannelId)

	//只解析HeaderType = HeaderType_ENDORSER_TRANSACTION
	if common.HeaderType(channelHeader.Type) != common.HeaderType_ENDORSER_TRANSACTION {
		log.Println("ignore other tx type",
			"headerType",
			channelHeader.Type,
		)
		return nil
	}

	signatureHeader, err := UnmarshalSignatureHeader(payload.Header.SignatureHeader)
	if err != nil {
		return err
	}

	singer, err := UnmarshalSerializedIdentity(signatureHeader.Creator)
	if err != nil {
		return err
	}

	//解析ENDORSER_TRANSACTION类型交易
	tx, err := UnmarshalTransaction(payload.Data)
	if err != nil {
		return err
	}

	if len(tx.Actions) == 0 {
		return errors.New("at least one TransactionAction required")
	}

	//获取交易结果
	_, chaincodeAction, err := GetPayloads(tx.Actions[0])
	if err != nil {
		return errors.New(err.Error())
	}
	//忽略非指定的链码调用
	if chaincodeAction == nil {
		log.Println("chaincodeaction is nil")
		return nil
	}
	log.Println("chaincode is:", chaincodeAction.ChaincodeId.Name)

	//忽略失败的chaincode调用
	if chaincodeAction.Response.Status != 200 {
		log.Println("ignore failed chaincode invocation")
		return nil
	}

	//解析读写集内容
	txRWSet := &rwsetutil.TxRwSet{}
	err = txRWSet.FromProtoBytes(chaincodeAction.Results)
	if err != nil {
		return err
	}

	for _, nsRWSet := range txRWSet.NsRwSets {
		log.Println("nsRWSet.NameSpace:",nsRWSet.NameSpace)
		for _, kvWrite := range nsRWSet.KvRwSet.Writes {
			if len(kvWrite.Value) == 0 {
				continue
			}

			log.Println("key is:",kvWrite.Key)
			log.Println("value is:", string(kvWrite.Value))
			log.Println("Mspid is:", singer.Mspid)
			log.Println("TxId is:", channelHeader.TxId)
			log.Println("Timestamp is:", channelHeader.Timestamp.GetSeconds())
		}
	}
	return nil

}
func GetEnvelopeFromBlock(data []byte) (*common.Envelope, error) {
	// Block always begins with an envelope
	var err error
	env := &common.Envelope{}
	if err = proto.Unmarshal(data, env); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling Envelope")
	}

	return env, nil
}

func  UnmarshalPayload(encoded []byte) (*common.Payload, error) {
	payload := &common.Payload{}
	err := proto.Unmarshal(encoded, payload)
	return payload, errors.Wrap(err, "error unmarshaling Payload")
}
func UnmarshalChannelHeader(bytes []byte) (*cb.ChannelHeader, error) {
	chdr := &cb.ChannelHeader{}
	err := proto.Unmarshal(bytes, chdr)
	return chdr, errors.Wrap(err, "error unmarshaling ChannelHeader")
}
func UnmarshalSignatureHeader(bytes []byte) (*cb.SignatureHeader, error) {
	sh := &common.SignatureHeader{}
	err := proto.Unmarshal(bytes, sh)
	return sh, errors.Wrap(err, "error unmarshaling SignatureHeader")
}
func UnmarshalSignatureHeaderOrPanic(bytes []byte) *cb.SignatureHeader {
	sighdr, err := UnmarshalSignatureHeader(bytes)
	if err != nil {
		panic(err)
	}
	return sighdr
}

func UnmarshalSerializedIdentity(bytes []byte) (*msp.SerializedIdentity, error) {
	sid := &msp.SerializedIdentity{}
	err := proto.Unmarshal(bytes, sid)
	return sid, errors.Wrap(err, "error unmarshaling SerializedIdentity")
}

func UnmarshalTransaction(txBytes []byte) (*peer.Transaction, error) {
	tx := &peer.Transaction{}
	err := proto.Unmarshal(txBytes, tx)
	return tx, errors.Wrap(err, "error unmarshaling Transaction")
}

func GetPayloads(txActions *peer.TransactionAction) (*peer.ChaincodeActionPayload, *peer.ChaincodeAction, error) {
	// TODO: pass in the tx type (in what follows we're assuming the
	// type is ENDORSER_TRANSACTION)
	ccPayload, err := UnmarshalChaincodeActionPayload(txActions.Payload)
	if err != nil {
		return nil, nil, err
	}

	if ccPayload.Action == nil || ccPayload.Action.ProposalResponsePayload == nil {
		return nil, nil, errors.New("no payload in ChaincodeActionPayload")
	}

	pRespPayload, err := UnmarshalProposalResponsePayload(ccPayload.Action.ProposalResponsePayload)
	if err != nil {
		return nil, nil, err
	}

	if pRespPayload.Extension == nil {
		return nil, nil, errors.New("response payload is missing extension")
	}

	respPayload, err := UnmarshalChaincodeAction(pRespPayload.Extension)
	if err != nil {
		return ccPayload, nil, err
	}
	return ccPayload, respPayload, nil
}

func UnmarshalChaincodeActionPayload(capBytes []byte) (*peer.ChaincodeActionPayload, error) {
	cap := &peer.ChaincodeActionPayload{}
	err := proto.Unmarshal(capBytes, cap)
	return cap, errors.Wrap(err, "error unmarshaling ChaincodeActionPayload")
}

func UnmarshalProposalResponsePayload(prpBytes []byte) (*peer.ProposalResponsePayload, error) {
	prp := &peer.ProposalResponsePayload{}
	err := proto.Unmarshal(prpBytes, prp)
	return prp, errors.Wrap(err, "error unmarshaling ProposalResponsePayload")
}

func UnmarshalChaincodeAction(caBytes []byte) (*peer.ChaincodeAction, error) {
	chaincodeAction := &peer.ChaincodeAction{}
	err := proto.Unmarshal(caBytes, chaincodeAction)
	return chaincodeAction, errors.Wrap(err, "error unmarshaling ChaincodeAction")
}
