package testutil

import (
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/suite"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/cosmos/cosmos-sdk/codec/testdata"

	"github.com/cosmos/cosmos-sdk/client"

	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TxGeneratorTestSuite provides a test suite that can be used to test that a TxGenerator implementation is correct
//nolint:golint  // type name will be used as tx.TxGeneratorTestSuite by other packages, and that stutters; consider calling this GeneratorTestSuite
type TxGeneratorTestSuite struct {
	suite.Suite
	TxGenerator client.TxGenerator
}

// NewTxGeneratorTestSuite returns a new TxGeneratorTestSuite with the provided TxGenerator implementation
func NewTxGeneratorTestSuite(txGenerator client.TxGenerator) *TxGeneratorTestSuite {
	return &TxGeneratorTestSuite{TxGenerator: txGenerator}
}

func (s *TxGeneratorTestSuite) TestTxBuilderGetTx() {
	txBuilder := s.TxGenerator.NewTxBuilder()
	tx := txBuilder.GetTx()
	s.Require().NotNil(tx)
	s.Require().Equal(len(tx.GetMsgs()), 0)
}

func (s *TxGeneratorTestSuite) TestTxBuilderSetFeeAmount() {
	txBuilder := s.TxGenerator.NewTxBuilder()
	feeAmount := sdk.Coins{
		sdk.NewInt64Coin("atom", 20000000),
	}
	txBuilder.SetFeeAmount(feeAmount)
	feeTx := txBuilder.GetTx()
	s.Require().Equal(feeAmount, feeTx.GetFee())
}

func (s *TxGeneratorTestSuite) TestTxBuilderSetGasLimit() {
	const newGas uint64 = 300000
	txBuilder := s.TxGenerator.NewTxBuilder()
	txBuilder.SetGasLimit(newGas)
	feeTx := txBuilder.GetTx()
	s.Require().Equal(newGas, feeTx.GetGas())
}

func (s *TxGeneratorTestSuite) TestTxBuilderSetMemo() {
	const newMemo string = "newfoomemo"
	txBuilder := s.TxGenerator.NewTxBuilder()
	txBuilder.SetMemo(newMemo)
	txWithMemo := txBuilder.GetTx()
	s.Require().Equal(txWithMemo.GetMemo(), newMemo)
}

func (s *TxGeneratorTestSuite) TestTxBuilderSetMsgs() {
	addr1 := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	addr2 := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	msg1 := testdata.NewTestMsg(addr1)
	msg2 := testdata.NewTestMsg(addr2)
	msgs := []sdk.Msg{msg1, msg2}

	txBuilder := s.TxGenerator.NewTxBuilder()

	err := txBuilder.SetMsgs(msgs...)
	s.Require().NoError(err)
	s.Require().Equal(msgs, txBuilder.GetTx().GetMsgs())
}

func (s *TxGeneratorTestSuite) TestTxBuilderSetSignatures() {
	privKey, pubkey, addr := authtypes.KeyTestPubAddr()

	txBuilder := s.TxGenerator.NewTxBuilder()

	// set test msg
	msg := testdata.NewTestMsg(addr)
	err := txBuilder.SetMsgs(msg)
	s.Require().NoError(err)

	// set SignatureV2 without actual signature bytes
	signModeHandler := s.TxGenerator.SignModeHandler()
	s.Require().Contains(signModeHandler.Modes(), signModeHandler.DefaultMode())
	sigData := &signingtypes.SingleSignatureData{SignMode: signModeHandler.DefaultMode()}
	sig := signingtypes.SignatureV2{PubKey: pubkey, Data: sigData}
	err = txBuilder.SetSignatures(sig)
	s.Require().NoError(err)
	sigTx := txBuilder.GetTx().(signing.SigVerifiableTx)
	s.Require().Equal([][]byte{nil}, sigTx.GetSignatures())
	sigsV2, err := sigTx.GetSignaturesV2()
	s.Require().NoError(err)
	s.Require().Equal([]signingtypes.SignatureV2{sig}, sigsV2)

	// sign transaction
	signBytes, err := signModeHandler.GetSignBytes(signModeHandler.DefaultMode(), signing.SignerData{
		ChainID:         "test",
		AccountNumber:   1,
		AccountSequence: 2,
	}, sigTx)
	s.Require().NoError(err)
	sigBz, err := privKey.Sign(signBytes)
	s.Require().NoError(err)

	// set signature
	sigData.Signature = sigBz
	sig = signingtypes.SignatureV2{PubKey: pubkey, Data: sigData}
	err = txBuilder.SetSignatures(sig)
	s.Require().NoError(err)
	sigTx = txBuilder.GetTx().(signing.SigVerifiableTx)
	s.Require().Equal([][]byte{nil}, sigTx.GetSignatures())
	sigsV2, err = sigTx.GetSignaturesV2()
	s.Require().NoError(err)
	s.Require().Equal([]signingtypes.SignatureV2{sig}, sigsV2)
}

func (s *TxGeneratorTestSuite) TestTxEncodeDecode() {
	priv := secp256k1.GenPrivKey()
	feeAmount := sdk.Coins{sdk.NewInt64Coin("atom", 150)}
	gasLimit := uint64(50000)
	memo := "foomemo"
	msg := testdata.NewTestMsg(sdk.AccAddress(priv.PubKey().Address()))
	dummySig := []byte("dummySig")
	sig := signingtypes.SignatureV2{
		PubKey: priv.PubKey(),
		Data: &signingtypes.SingleSignatureData{
			SignMode:  signingtypes.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
			Signature: dummySig,
		},
	}

	txBuilder := s.TxGenerator.NewTxBuilder()
	txBuilder.SetFeeAmount(feeAmount)
	txBuilder.SetGasLimit(gasLimit)
	txBuilder.SetMemo(memo)
	err := txBuilder.SetMsgs(msg)
	s.Require().NoError(err)
	err = txBuilder.SetSignatures(sig)
	s.Require().NoError(err)
	tx := txBuilder.GetTx()

	// encode transaction
	txBytes, err := s.TxGenerator.TxEncoder()(tx)
	s.Require().NoError(err)
	s.Require().NotNil(txBytes)

	// decode transaction
	tx2, err := s.TxGenerator.TxDecoder()(txBytes)
	s.Require().NoError(err)
	tx3, ok := tx2.(signing.SigFeeMemoTx)
	s.Require().True(ok)
	s.Require().Equal([]sdk.Msg{msg}, tx3.GetMsgs())
	s.Require().Equal(feeAmount, tx3.GetFee())
	s.Require().Equal(gasLimit, tx3.GetGas())
	s.Require().Equal(memo, tx3.GetMemo())
	s.Require().Equal(dummySig, tx3.GetSignatures()[0])
	s.Require().Equal(priv.PubKey(), tx3.GetPubKeys()[0])

	// JSON encode transaction
	jsonTxBytes, err := s.TxGenerator.TxJSONEncoder()(tx)
	s.Require().NoError(err)
	s.Require().NotNil(jsonTxBytes)

	// JSON decode transaction
	tx2, err = s.TxGenerator.TxJSONDecoder()(txBytes)
	s.Require().NoError(err)
	tx3, ok = tx2.(signing.SigFeeMemoTx)
	s.Require().True(ok)
	s.Require().Equal([]sdk.Msg{msg}, tx3.GetMsgs())
	s.Require().Equal(feeAmount, tx3.GetFee())
	s.Require().Equal(gasLimit, tx3.GetGas())
	s.Require().Equal(memo, tx3.GetMemo())
	s.Require().Equal(dummySig, tx3.GetSignatures()[0])
	s.Require().Equal(priv.PubKey(), tx3.GetPubKeys()[0])
}
