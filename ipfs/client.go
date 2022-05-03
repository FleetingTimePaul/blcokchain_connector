package ipfs

import (
	"bytes"
	"github.com/golang/glog"
	shell "github.com/ipfs/go-ipfs-api"
	"io/ioutil"
)

var ipfsClient *Client

type Client struct {
	ipfsNodeUrl string
}

func InitClient(url string) *Client {
	c := &Client{
		ipfsNodeUrl: url,
	}

	return c
}

func (c *Client) UploadContent(content string) (string, error) {
	sh := shell.NewShell(c.ipfsNodeUrl)
	hash, err := sh.Add(bytes.NewBufferString(content))
	if err != nil {
		glog.Errorf("ipfs UploadContent :%+v", err)
		return "", err
	}
	return hash, err
}

func (c *Client) DownloadContent(hash string) (string, error) {
	sh := shell.NewShell(c.ipfsNodeUrl)
	read, err := sh.Cat(hash)
	if err != nil {
		glog.Errorf("ipfs DownloadContent :%+v", err)
		return "", err
	}
	body, err := ioutil.ReadAll(read)
	if err != nil {
		glog.Errorf("ipfs DownloadContent :%+v", err)
		return "", err
	}
	return string(body), nil
}
