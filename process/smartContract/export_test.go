package smartContract

import (
	"math/big"

	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go/data/state"
	"github.com/ElrondNetwork/elrond-go/data/transaction"
	"github.com/ElrondNetwork/elrond-vm-common"
)

func (sc *scProcessor) CreateVMCallInput(tx *transaction.Transaction) (*vmcommon.ContractCallInput, error) {
	return sc.createVMCallInput(tx)
}

func (sc *scProcessor) CreateVMDeployInput(tx *transaction.Transaction) (*vmcommon.ContractCreateInput, []byte, error) {
	return sc.createVMDeployInput(tx)
}

func (sc *scProcessor) CreateVMInput(tx *transaction.Transaction) (*vmcommon.VMInput, error) {
	return sc.createVMInput(tx)
}

func (sc *scProcessor) ProcessVMOutput(
	vmOutput *vmcommon.VMOutput,
	tx *transaction.Transaction,
	acntSnd state.AccountHandler,
	round uint64,
) ([]data.TransactionHandler, *big.Int, error) {
	return sc.processVMOutput(vmOutput, tx, acntSnd, round)
}

func (sc *scProcessor) CreateSCRForSender(
	vmOutput *vmcommon.VMOutput,
	tx *transaction.Transaction,
	txHash []byte,
	acntSnd state.AccountHandler,
) (*smartContractResult.SmartContractResult, *big.Int, error) {
	return sc.createSCRForSender(vmOutput, tx, txHash, acntSnd)
}

func (sc *scProcessor) ProcessSCOutputAccounts(outputAccounts []*vmcommon.OutputAccount, tx *transaction.Transaction) error {
	return sc.processSCOutputAccounts(outputAccounts, tx)
}

func (sc *scProcessor) DeleteAccounts(deletedAccounts [][]byte) error {
	return sc.deleteAccounts(deletedAccounts)
}

func (sc *scProcessor) GetAccountFromAddress(address []byte) (state.AccountHandler, error) {
	return sc.getAccountFromAddress(address)
}

func (sc *scProcessor) SaveSCOutputToCurrentState(output *vmcommon.VMOutput, round uint64, txHash []byte) error {
	return sc.saveSCOutputToCurrentState(output, round, txHash)
}

func (sc *scProcessor) SaveReturnData(returnData [][]byte, round uint64, txHash []byte) error {
	return sc.saveReturnData(returnData, round, txHash)
}

func (sc *scProcessor) SaveReturnCode(returnCode vmcommon.ReturnCode, round uint64, txHash []byte) error {
	return sc.saveReturnCode(returnCode, round, txHash)
}

func (sc *scProcessor) SaveLogsIntoState(logs []*vmcommon.LogEntry, round uint64, txHash []byte) error {
	return sc.saveLogsIntoState(logs, round, txHash)
}

func (sc *scProcessor) ProcessSCPayment(tx *transaction.Transaction, acntSnd state.AccountHandler) error {
	return sc.processSCPayment(tx, acntSnd)
}

func (sc *scProcessor) CreateSCRTransactions(
	crossOutAccs []*vmcommon.OutputAccount,
	tx *transaction.Transaction,
	txHash []byte,
) ([]data.TransactionHandler, error) {
	return sc.createSCRTransactions(crossOutAccs, tx, txHash)
}
