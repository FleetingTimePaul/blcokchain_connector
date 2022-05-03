package arweave

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"github.com/golang/glog"
	"io/ioutil"
	"os"
	"time"
)

var arClient *Client

type Client struct {
	arUrl   string
	proxy   string
	keyPath string
	wallet  *goar.Wallet
}

func (c *Client) initClient() {
	var err error
	c.wallet, err = goar.NewWalletFromPath(c.keyPath, c.arUrl)
	if err != nil {
		glog.Errorf("init arclient err:%+v", err)
		return
	}
}

func CreateClient(arUrl, proxy, keyPath string) *Client {
	arClient := &Client{
		arUrl:   arUrl,
		proxy:   proxy,
		keyPath: keyPath,
	}

	arClient.initClient()
	return arClient
}

func (c *Client) assemblyDataTx(bigData []byte, tags []types.Tag) (*types.Transaction, error) {
	wallet := c.wallet
	reward, err := wallet.Client.GetTransactionPrice(bigData, nil)
	if err != nil {
		glog.Errorf("assemblyDataTx err:%+v", err)
		return nil, err
	}
	tx := &types.Transaction{
		Format:   2,
		Target:   "",
		Quantity: "0",
		Tags:     utils.TagsEncode(tags),
		Data:     utils.Base64Encode(bigData),
		DataSize: fmt.Sprintf("%d", len(bigData)),
		Reward:   fmt.Sprintf("%d", reward),
	}
	anchor, err := wallet.Client.GetTransactionAnchor()
	if err != nil {
		glog.Errorf("assemblyDataTx err:%+v", err)
		return nil, err
	}
	tx.LastTx = anchor
	tx.Owner = wallet.Owner()

	signData, err := utils.GetSignatureData(tx)
	if err != nil {
		glog.Errorf("assemblyDataTx err:%+v", err)
		return nil, err
	}

	sign, err := wallet.Signer.SignMsg(signData)
	if err != nil {
		glog.Errorf("assemblyDataTx err:%+v", err)
		return nil, err
	}

	txHash := sha256.Sum256(sign)
	tx.ID = utils.Base64Encode(txHash[:])

	tx.Signature = utils.Base64Encode(sign)
	return tx, nil
}

func (c *Client) UploadContent(content string, tags []types.Tag) (string, error) {
	var err error
	wallet := c.wallet
	bigData := []byte(content)
	//tags := []types.Tag{{Name: "Content-Type", Value: "application/jpg"}, {Name: "goar", Value: "1.8mbPhoto"}}
	tx, err := c.assemblyDataTx(bigData, tags)

	// 1. Upload a portion of data, for test when uploaded chunk == 2 and stop upload
	uploader, err := goar.CreateUploader(wallet.Client, tx, nil)
	if err != nil {
		glog.Errorf("ar UploadContent err:%+v", err)
		return "", err
	}

	// only upload 2 chunks to ar chain
	for !uploader.IsComplete() && uploader.ChunkIndex <= 2 {
		err = uploader.UploadChunk()
		if err != nil {
			glog.Errorf("ar UploadContent err:%+v", err)
			return "", err
		}
	}

	// then store uploader object to file
	jsonUploader, err := json.Marshal(uploader)
	if err != nil {
		glog.Errorf("ar UploadContent err:%+v", err)
		return "", err
	}

	err = ioutil.WriteFile("./jsonUploaderFile.json", jsonUploader, 0777)
	if err != nil {
		glog.Errorf("ar UploadContent err:%+v", err)
		return "", err
	}

	time.Sleep(5 * time.Second) // sleep 5s

	// 2. read uploader object from jsonUploader.json file and continue upload by last time uploader
	uploaderBuf, err := ioutil.ReadFile("./jsonUploaderFile.json")
	if err != nil {
		glog.Errorf("ar UploadContent err:%+v", err)
		return "", err
	}

	lastUploader := &goar.TransactionUploader{}
	err = json.Unmarshal(uploaderBuf, lastUploader)
	if err != nil {
		glog.Errorf("ar UploadContent err:%+v", err)
		return "", err
	}

	// new uploader object by last time uploader
	newUploader, err := goar.CreateUploader(wallet.Client, lastUploader.FormatSerializedUploader(), bigData)
	if err != nil {
		glog.Errorf("ar UploadContent err:%+v", err)
		return "", err
	}

	err = newUploader.Once()
	if err != nil {
		glog.Errorf("ar UploadContent err:%+v", err)
		return "", err
	}

	// end remove jsonUploaderFile.json file
	err = os.Remove("./jsonUploaderFile.json")
	if err != nil {
		glog.Errorf("ar UploadContent err:%+v", err)
		return "", err
	}

	return tx.ID, nil
}

func (c *Client) DownloadContent(id string, extension ...string) (string, error) {
	wallet := c.wallet
	body, err := wallet.Client.GetTransactionData(id, extension...)
	if err != nil {
		glog.Errorf("ar DownloadContent err:%+v", err)
		return "", err
	}
	return string(body), nil
}

func (c *Client) GetTags(id string) (string, error) {
	wallet := c.wallet
	tags, err := wallet.Client.GetTransactionTags(id)
	if err != nil {
		glog.Errorf("ar GetTags err:%+v", err)
		return "", err
	}

	body, err := json.Marshal(tags)
	if err != nil {
		glog.Errorf("ar GetTags err:%+v", err)
		return "", err
	}

	return string(body), nil
}
