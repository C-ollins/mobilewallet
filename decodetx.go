package dcrlibwallet

import (
	"encoding/binary"
	"fmt"

	"github.com/decred/dcrd/blockchain/stake/v2"
	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/decred/dcrd/txscript/v2"
	"github.com/decred/dcrd/wire"
	"github.com/decred/dcrwallet/wallet/v3"
	"github.com/raedahgroup/dcrlibwallet/txhelper"
)

const (
	BlockHeightInvalid int32 = -1
	BlockValid               = 1 << 0
)

// DecodeTransaction uses `walletTx.Hex` to retrieve detailed information for a transaction.
func DecodeTransaction(walletTx *TxInfoFromWallet, netParams *chaincfg.Params) (*Transaction, error) {
	msgTx, txFee, txSize, txFeeRate, err := txhelper.MsgTxFeeSizeRate(walletTx.Hex)
	if err != nil {
		return nil, err
	}
	txType := wallet.TxTransactionType(msgTx)

	// only use input/output amounts relating to wallet to correctly determine tx direction
	var totalWalletInput, totalWalletOutput int64
	for _, input := range walletTx.Inputs {
		totalWalletInput += input.AmountIn
	}
	for _, output := range walletTx.Outputs {
		totalWalletOutput += output.AmountOut
	}
	amount, direction := txhelper.TransactionAmountAndDirection(totalWalletInput, totalWalletOutput, int64(txFee))

	inputs := decodeTxInputs(msgTx, walletTx.Inputs)
	outputs := decodeTxOutputs(msgTx, netParams, walletTx.Outputs)

	ssGenVersion, lastBlockValid, voteBits := voteInfo(msgTx)

	return &Transaction{
		WalletID:    walletTx.WalletID,
		Hash:        msgTx.TxHash().String(),
		Type:        txhelper.FormatTransactionType(txType),
		Hex:         walletTx.Hex,
		Timestamp:   walletTx.Timestamp,
		BlockHeight: walletTx.BlockHeight,

		Version:  int32(msgTx.Version),
		LockTime: int32(msgTx.LockTime),
		Expiry:   int32(msgTx.Expiry),
		Fee:      int64(txFee),
		FeeRate:  int64(txFeeRate),
		Size:     txSize,

		Direction: direction,
		Amount:    amount,
		Inputs:    inputs,
		Outputs:   outputs,

		VoteVersion:    int32(ssGenVersion),
		LastBlockValid: lastBlockValid,
		VoteBits:       voteBits,
	}, nil
}

func decodeTxInputs(mtx *wire.MsgTx, walletInputs []*WalletInput) (inputs []*TxInput) {
	inputs = make([]*TxInput, len(mtx.TxIn))

	for i, txIn := range mtx.TxIn {
		input := &TxInput{
			PreviousTransactionHash:  txIn.PreviousOutPoint.Hash.String(),
			PreviousTransactionIndex: int32(txIn.PreviousOutPoint.Index),
			PreviousOutpoint:         txIn.PreviousOutPoint.String(),
			Amount:                   txIn.ValueIn,
			AccountName:              "external", // correct account name and number set below if this is a wallet output
			AccountNumber:            -1,
		}

		// override account details if this is wallet input
		for _, walletInput := range walletInputs {
			if walletInput.Index == int32(i) {
				input.AccountName = walletInput.AccountName
				input.AccountNumber = walletInput.AccountNumber
				break
			}
		}

		inputs[i] = input
	}

	return
}

func decodeTxOutputs(mtx *wire.MsgTx, netParams *chaincfg.Params, walletOutputs []*WalletOutput) (outputs []*TxOutput) {
	outputs = make([]*TxOutput, len(mtx.TxOut))
	txType := stake.DetermineTxType(mtx)

	for i, txOut := range mtx.TxOut {
		// get address and script type for output
		var address, scriptType string
		if (txType == stake.TxTypeSStx) && (stake.IsStakeSubmissionTxOut(i)) {
			addr, err := stake.AddrFromSStxPkScrCommitment(txOut.PkScript, netParams)
			if err == nil {
				address = addr.Address()
			}
			scriptType = txscript.StakeSubmissionTy.String()
		} else {
			// Ignore the error here since an error means the script
			// couldn't parse and there is no additional information
			// about it anyways.
			scriptClass, addrs, _, _ := txscript.ExtractPkScriptAddrs(txOut.Version, txOut.PkScript, netParams)
			if len(addrs) > 0 {
				address = addrs[0].Address()
			}
			scriptType = scriptClass.String()
		}

		output := &TxOutput{
			Index:         int32(i),
			Amount:        txOut.Value,
			Version:       int32(txOut.Version),
			ScriptType:    scriptType,
			Address:       address, // correct address, account name and number set below if this is a wallet output
			AccountName:   "external",
			AccountNumber: -1,
		}

		// override address and account details if this is wallet output
		for _, walletOutput := range walletOutputs {
			if walletOutput.Index == output.Index {
				output.Internal = walletOutput.Internal
				output.Address = walletOutput.Address
				output.AccountName = walletOutput.AccountName
				output.AccountNumber = walletOutput.AccountNumber
				break
			}
		}

		outputs[i] = output
	}

	return
}

func voteInfo(msgTx *wire.MsgTx) (ssGenVersion uint32, lastBlockValid bool, voteBits string) {
	if stake.IsSSGen(msgTx) {
		ssGenVersion = voteVersion(msgTx)
		bits := binary.LittleEndian.Uint16(msgTx.TxOut[1].PkScript[2:4])
		voteBits = fmt.Sprintf("%#04x", bits)
		lastBlockValid = bits&uint16(BlockValid) != 0
	}
	return
}

func voteVersion(mtx *wire.MsgTx) uint32 {
	if len(mtx.TxOut[1].PkScript) < 8 {
		return 0 // Consensus version absent
	}

	return binary.LittleEndian.Uint32(mtx.TxOut[1].PkScript[4:8])
}
