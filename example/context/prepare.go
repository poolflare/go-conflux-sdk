package context

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/BurntSushi/toml"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	exampletypes "github.com/Conflux-Chain/go-conflux-sdk/example/context/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var (
	config         exampletypes.Config
	client         *sdk.Client
	currentDir     string
	configPath     string
	am             *sdk.AccountManager
	defaultAccount *types.Address
	nextNonce      *big.Int
)

func PrepareForClientExample() *exampletypes.Config {
	fmt.Println("=======start prepare config===========\n")
	getConfig()
	initClient()
	generateBlockHashAndTxHash()
	deployContract()
	saveConfig()
	fmt.Println("=======prepare config done!===========\n")
	return &config
}

func PrepareForContractExample() *exampletypes.Config {
	fmt.Println("=======start prepare config===========\n")
	getConfig()
	initClient()
	saveConfig()
	fmt.Println("=======prepare config done!===========\n")
	return &config
}

func getConfig() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("get current file path error")
	}
	currentDir = path.Join(filename, "../")
	configPath = path.Join(currentDir, "config.toml")
	// cp := make(map[string]string)
	config = exampletypes.Config{}
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("- to get config done: %+v\n", JsonFmt(config))
}

func initClient() {
	// url := "http://testnet-jsonrpc.conflux-chain.org:12537"
	var err error
	client, err = sdk.NewClient(config.NodeURL)
	if err != nil {
		panic(err)
	}
	config.SetClient(client)

	retryclient, err := sdk.NewClientWithRetry(config.NodeURL, 10, time.Second)
	if err != nil {
		panic(err)
	}
	config.SetRetryClient(retryclient)

	am = sdk.NewAccountManager(path.Join(currentDir, "keystore"))
	// fmt.Printf("am in preapre:%v", am)
	client.SetAccountManager(am)
	defaultAccount, err = am.GetDefault()
	if err != nil {
		panic(err)
	}
	am.UnlockDefault("hello")
	config.SetAccountManager(am)

	nextNonce, err = client.GetNextNonce(*defaultAccount, nil)
	if err != nil {
		panic(err)
	}

	fmt.Println("- to init client done")
}

func generateBlockHashAndTxHash() {

	block, err1 := client.GetBlockByHash(config.BlockHash)
	tx, err2 := client.GetTransactionByHash(config.TransactionHash)
	if block == nil || err1 != nil || tx == nil || err2 != nil {
		utx, err := client.CreateUnsignedTransaction(*defaultAccount, *types.NewAddress("0x10f4bcf113e0b896d9b34294fd3da86b4adf0302"), types.NewBigInt(1), nil)
		if err != nil {
			panic(err)
		}
		utx.Nonce = getNextNonceAndIncrease()
		txhash, err := client.SendTransaction(utx)
		if err != nil {
			panic(err)
		}
		config.TransactionHash = txhash

		WaitPacked(client, txhash)

		tx, err := client.GetTransactionByHash(txhash)
		if err != nil {
			panic(err)
		}
		config.BlockHash = *tx.BlockHash
	}

	fmt.Println("- gen txhash done")
}

func deployContract() {
	// check erc20 and erc777 address, if len !==42 or getcode error, deploy
	erc20Contract := DeployIfNotExist(config.ERC20Address, path.Join(currentDir, "contract/erc20.abi"), path.Join(currentDir, "contract/erc20.bytecode"))
	if erc20Contract != nil {
		config.ERC20Address = *erc20Contract.Address
	}
	fmt.Println("- to deploy contracts if not exist done")

}

func saveConfig() {
	f, err := os.OpenFile(configPath, os.O_RDWR, os.ModePerm)
	if err != nil {
		panic(err)
	}
	encoder := toml.NewEncoder(f)
	err = encoder.Encode(config)
	if err != nil {
		panic(err)
	}
	fmt.Println("- to save config done")
}

func getNextNonceAndIncrease() *hexutil.Big {
	// println("current in:", nextNonce.String())
	currentNonce := big.NewInt(0).SetBytes(nextNonce.Bytes())
	nextNonce = nextNonce.Add(nextNonce, big.NewInt(1))
	// println("current out:", currentNonce.String())
	return types.NewBigIntByRaw(currentNonce)
}
