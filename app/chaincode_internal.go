package main

import (
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode"
	"github.com/hyperledger/fabric/core/chaincode/platforms"
	"github.com/hyperledger/fabric/core/container"
	"github.com/hyperledger/fabric/core/crypto"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/util"
	pb "github.com/hyperledger/fabric/protos"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
)

var (
	confidentialityOn    bool
	adminInvoker         crypto.Client
	confidentialityLevel pb.ConfidentialityLevel
)

func initNVP() (err error) {
	if err = initPeerClient(); err != nil {
		// myLogger.Debugf("Failed initNVP [%s]", err)
		return
	}

	adminInvoker, err = setCryptoClient("admin", "Xurw3yU9zI0l")
	if err != nil {
		// myLogger.Errorf("Failed getting invoker [%s]", err)
		return
	}

	return
}

func initPeerClient() (err error) {
	peerClientConn, err = peer.NewPeerClientConnection()
	if err != nil {
		fmt.Printf("error connection to server at host:port = %s\n", viper.GetString("peer.address"))
		return
	}
	serverClient = pb.NewPeerClient(peerClientConn)

	// Logging
	var formatter = logging.MustStringFormatter(
		`%{color}[%{module}] %{shortfunc} [%{shortfile}] -> %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)
	logging.SetFormatter(formatter)

	return
}

func setCryptoClient(enrollID, enrollPWD string) (crypto.Client, error) {
	if len(enrollPWD) > 0 {
		if err := crypto.RegisterClient(enrollID, nil, enrollID, enrollPWD); err != nil {
			return nil, err
		}
	}

	client, err := crypto.InitClient(enrollID, nil)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func processTransaction(tx *pb.Transaction) (*pb.Response, error) {
	return serverClient.ProcessTransaction(context.Background(), tx)
}

func processTransaction2(tx *pb.Transaction, client pb.PeerClient) (*pb.Response, error) {
	return client.ProcessTransaction(context.Background(), tx)
}

func confidentiality(enabled bool) {
	confidentialityOn = enabled

	if confidentialityOn {
		confidentialityLevel = pb.ConfidentialityLevel_CONFIDENTIAL
	} else {
		confidentialityLevel = pb.ConfidentialityLevel_PUBLIC
	}
}

func deployInternal() (resp *pb.Response, err error) {
	chaincodePath = viper.GetString("chaincode.id.path")
	// Prepare the spec
	spec := &pb.ChaincodeSpec{
		Type:        pb.ChaincodeSpec_GOLANG,
		ChaincodeID: &pb.ChaincodeID{Path: chaincodePath},
		CtorMsg:     &pb.ChaincodeInput{Args: util.ToChaincodeArgs("init")},
		// Metadata:             "",
		ConfidentialityLevel: confidentialityLevel,
	}

	// First build the deployment spec
	cds, err := getChaincodeBytes(spec)
	if err != nil {
		return nil, fmt.Errorf("Error getting deployment spec: %s ", err)
	}

	// Now create the Transactions message and send to Peer.
	transaction, err := adminInvoker.NewChaincodeDeployTransaction(cds, cds.ChaincodeSpec.ChaincodeID.Name)
	if err != nil {
		return nil, fmt.Errorf("Error deploying chaincode: %s ", err)
	}

	resp, err = processTransaction(transaction)

	// myLogger.Debugf("resp [%s]", resp.String())

	chaincodeName = cds.ChaincodeSpec.ChaincodeID.Name
	// myLogger.Debugf("ChaincodeName [%s]", chaincodeName)

	return
}

func invokeChaincodeSigma(invoker crypto.Client, invokerCert crypto.CertificateHandler, chaincodeInput *pb.ChaincodeInput) (result string, err error) {
	// myLogger.Debug("------------- invoke...")
	// Get a transaction handler to be used to submit the execute transaction
	// and bind the chaincode access control logic using the binding
	submittingCertHandler, err := invoker.GetTCertificateHandlerNext()
	if err != nil {
		return "", fmt.Errorf("Error invoking chaincode: %s ", err)
	}
	txHandler, err := submittingCertHandler.GetTransactionHandler()
	if err != nil {
		return "", fmt.Errorf("Error invoking chaincode: %s ", err)
	}
	binding, err := txHandler.GetBinding()
	if err != nil {
		return "", fmt.Errorf("Error invoking chaincode: %s ", err)
	}

	chaincodeInputRaw, err := proto.Marshal(chaincodeInput)
	if err != nil {
		return "", fmt.Errorf("Error invoking chaincode: %s ", err)
	}

	// Access control. Administrator signs chaincodeInputRaw || binding to confirm his identity
	sigma, err := invokerCert.Sign(append(chaincodeInputRaw, binding...))
	if err != nil {
		return "", fmt.Errorf("Error invoking chaincode: %s ", err)
	}

	// Prepare spec and submit
	spec := &pb.ChaincodeSpec{
		Type:                 pb.ChaincodeSpec_GOLANG,
		ChaincodeID:          &pb.ChaincodeID{Name: chaincodeName},
		CtorMsg:              chaincodeInput,
		Metadata:             sigma, // Proof of identity
		ConfidentialityLevel: confidentialityLevel,
	}

	chaincodeInvocationSpec := &pb.ChaincodeInvocationSpec{ChaincodeSpec: spec}

	// Now create the Transactions message and send to Peer.
	transaction, err := txHandler.NewChaincodeExecute(chaincodeInvocationSpec, util.GenerateUUID())
	if err != nil {
		return "", fmt.Errorf("Error invoking chaincode: %s ", err)
	}

	resp, err := processTransaction(transaction)
	if err != nil {
		return "", fmt.Errorf("Error invoking chaincode: %s ", err)
	}

	if resp.Status != pb.Response_SUCCESS {
		return "", fmt.Errorf("Error invoking chaincode: %s ", string(resp.Msg))
	}
	// myLogger.Debugf("Resp [%s]", resp.String())

	// myLogger.Debug("------------- Done!")

	return string(resp.Msg), nil
}

func invokeChaincode(invoker crypto.Client, chaincodeInput *pb.ChaincodeInput) (result string, err error) {
	// myLogger.Debug("------------- invoke...")
	// Get a transaction handler to be used to submit the execute transaction
	// and bind the chaincode access control logic using the binding
	submittingCertHandler, err := invoker.GetTCertificateHandlerNext()
	if err != nil {
		return "", fmt.Errorf("Error invoking chaincode: %s ", err)
	}
	txHandler, err := submittingCertHandler.GetTransactionHandler()
	if err != nil {
		return "", fmt.Errorf("Error invoking chaincode: %s ", err)
	}

	// Prepare spec and submit
	spec := &pb.ChaincodeSpec{
		Type:        pb.ChaincodeSpec_GOLANG,
		ChaincodeID: &pb.ChaincodeID{Name: chaincodeName},
		CtorMsg:     chaincodeInput,
		// Metadata:             sigma, // Proof of identity
		// SecureContext:        "admin",
		ConfidentialityLevel: confidentialityLevel,
	}

	chaincodeInvocationSpec := &pb.ChaincodeInvocationSpec{ChaincodeSpec: spec}

	// Now create the Transactions message and send to Peer.
	transaction, err := txHandler.NewChaincodeExecute(chaincodeInvocationSpec, util.GenerateUUID())
	if err != nil {
		return "", fmt.Errorf("Error invoking chaincode: %s ", err)
	}

	resp, err := processTransaction(transaction)
	if err != nil {
		return "", fmt.Errorf("Error invoking chaincode: %s ", err)
	}

	if resp.Status != pb.Response_SUCCESS {
		return "", fmt.Errorf("Error invoking chaincode: %s ", string(resp.Msg))
	}
	// myLogger.Debugf("Resp [%s]", resp.String())

	// myLogger.Debug("------------- Done!")

	return string(resp.Msg), nil
}

func queryChaincode(chaincodeInput *pb.ChaincodeInput) (result string, err error) {
	// myLogger.Debug("Query....")

	// Prepare spec and submit
	spec := &pb.ChaincodeSpec{
		Type:                 pb.ChaincodeSpec_GOLANG,
		ChaincodeID:          &pb.ChaincodeID{Name: chaincodeName},
		CtorMsg:              chaincodeInput,
		ConfidentialityLevel: confidentialityLevel,
	}

	chaincodeInvocationSpec := &pb.ChaincodeInvocationSpec{ChaincodeSpec: spec}

	// Now create the Transactions message and send to Peer.
	transaction, err := adminInvoker.NewChaincodeQuery(chaincodeInvocationSpec, util.GenerateUUID())
	if err != nil {
		return "", fmt.Errorf("Error query chaincode: %s ", err)
	}

	resp, err := processTransaction(transaction)

	// myLogger.Debugf("Resp [%s]", resp.String())
	// myLogger.Debug("Query....done")

	if resp.Status != pb.Response_SUCCESS {
		return "", errors.New(string(resp.Msg))
	}

	return string(resp.Msg), nil
}

func invokeChaincode2(invoker crypto.Client, chaincodeInput *pb.ChaincodeInput, serverClient pb.PeerClient) (result string, err error) {
	// myLogger.Debug("------------- invoke...")
	// Get a transaction handler to be used to submit the execute transaction
	// and bind the chaincode access control logic using the binding
	submittingCertHandler, err := invoker.GetTCertificateHandlerNext()
	if err != nil {
		return "", fmt.Errorf("Error invoking chaincode: %s ", err)
	}
	txHandler, err := submittingCertHandler.GetTransactionHandler()
	if err != nil {
		return "", fmt.Errorf("Error invoking chaincode: %s ", err)
	}

	// Prepare spec and submit
	spec := &pb.ChaincodeSpec{
		Type:        pb.ChaincodeSpec_GOLANG,
		ChaincodeID: &pb.ChaincodeID{Name: chaincodeName},
		CtorMsg:     chaincodeInput,
		// Metadata:             sigma, // Proof of identity
		ConfidentialityLevel: confidentialityLevel,
	}

	chaincodeInvocationSpec := &pb.ChaincodeInvocationSpec{ChaincodeSpec: spec}

	// Now create the Transactions message and send to Peer.
	transaction, err := txHandler.NewChaincodeExecute(chaincodeInvocationSpec, util.GenerateUUID())
	if err != nil {
		return "", fmt.Errorf("Error invoking chaincode: %s ", err)
	}

	resp, err := processTransaction2(transaction, serverClient)
	if err != nil {
		return "", fmt.Errorf("Error invoking chaincode: %s ", err)
	}

	if resp.Status != pb.Response_SUCCESS {
		return "", fmt.Errorf("Error invoking chaincode: %s ", string(resp.Msg))
	}
	// myLogger.Debugf("Resp [%s]", resp.String())

	// myLogger.Debug("------------- Done!")

	return string(resp.Msg), nil
}

func queryChaincode2(chaincodeInput *pb.ChaincodeInput, serverClient pb.PeerClient) (result string, err error) {
	// myLogger.Debug("Query....")

	// Prepare spec and submit
	spec := &pb.ChaincodeSpec{
		Type:                 pb.ChaincodeSpec_GOLANG,
		ChaincodeID:          &pb.ChaincodeID{Name: chaincodeName},
		CtorMsg:              chaincodeInput,
		ConfidentialityLevel: confidentialityLevel,
	}

	chaincodeInvocationSpec := &pb.ChaincodeInvocationSpec{ChaincodeSpec: spec}

	// Now create the Transactions message and send to Peer.
	transaction, err := adminInvoker.NewChaincodeQuery(chaincodeInvocationSpec, util.GenerateUUID())
	if err != nil {
		return "", fmt.Errorf("Error query chaincode: %s ", err)
	}

	resp, err := processTransaction2(transaction, serverClient)

	// myLogger.Debugf("Resp [%s]", resp.String())
	// myLogger.Debug("Query....done")

	if resp.Status != pb.Response_SUCCESS {
		return "", errors.New(string(resp.Msg))
	}

	return string(resp.Msg), nil
}

func getChaincodeBytes(spec *pb.ChaincodeSpec) (*pb.ChaincodeDeploymentSpec, error) {
	mode := viper.GetString("chaincode.mode")
	var codePackageBytes []byte
	if mode != chaincode.DevModeUserRunsChaincode {
		// myLogger.Debugf("Received build request for chaincode spec: %v", spec)
		var err error
		if err = checkSpec(spec); err != nil {
			return nil, err
		}

		codePackageBytes, err = container.GetChaincodePackageBytes(spec)
		if err != nil {
			err = fmt.Errorf("Error getting chaincode package bytes: %s", err)
			// myLogger.Errorf("%s", err)
			return nil, err
		}
	}
	chaincodeDeploymentSpec := &pb.ChaincodeDeploymentSpec{ChaincodeSpec: spec, CodePackage: codePackageBytes}
	return chaincodeDeploymentSpec, nil
}

func checkSpec(spec *pb.ChaincodeSpec) error {
	// Don't allow nil value
	if spec == nil {
		return errors.New("Expected chaincode specification, nil received")
	}

	platform, err := platforms.Find(spec.Type)
	if err != nil {
		return fmt.Errorf("Failed to determine platform type: %s", err)
	}

	return platform.ValidateSpec(spec)
}
