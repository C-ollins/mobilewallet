package txhelper

import (
	dcrutil "github.com/decred/dcrd/dcrutil/v2"
	"github.com/decred/dcrd/txscript"
	"github.com/decred/dcrd/wire"
	"github.com/raedahgroup/dcrlibwallet/addresshelper"
)

func MakeTxOutput(address string, amountInAtom int64, net dcrutil.AddressParams) (output *wire.TxOut, err error) {
	pkScript, err := addresshelper.PkScript(address, net)
	if err != nil {
		return
	}

	output = &wire.TxOut{
		Value:    amountInAtom,
		Version:  txscript.DefaultScriptVersion,
		PkScript: pkScript,
	}
	return
}
