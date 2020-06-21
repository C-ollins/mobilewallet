package main

import "github.com/raedahgroup/dcrlibwallet"

func main() {
	multiWallet, err := dcrlibwallet.NewMultiWallet("", "bdb", "testnet3")
	if err != nil{
		panic(err)
	}

	_, err = multiWallet.CreateNewWallet("", "c", 0)
	if err != nil{
		panic(err)
	}

	multiWallet.SetStringConfigValueForKey(dcrlibwallet.SpvPersistentPeerAddressesConfigKey, "127.0.0.1")

	err = multiWallet.SpvSync()
	if err != nil{
		panic(err)
	}

	select {}
}
