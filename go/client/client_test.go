package client

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/alxdavids/voprf-poc/go/jsonrpc"
	"github.com/alxdavids/voprf-poc/go/oerr"
	gg "github.com/alxdavids/voprf-poc/go/oprf/groups"
	"github.com/alxdavids/voprf-poc/go/oprf/groups/ecgroup"
	"github.com/stretchr/testify/assert"
)

var (
	validOPRFP384Ciphersuite = "OPRF-P384-HKDF-SHA512-SSWU-RO"
	validOPRFP521Ciphersuite = "OPRF-P521-HKDF-SHA512-SSWU-RO"
)

func TestCreateConfigP384(t *testing.T) {
	cfg, err := CreateConfig(validOPRFP384Ciphersuite, ecgroup.GroupCurve{}, 1, "some_file", -1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, cfg.ocli.Ciphersuite().POG().Name(), "P-384")
	assert.Equal(t, cfg.n, 1)
	assert.Equal(t, cfg.addr, "http://localhost:3001")
	assert.Equal(t, cfg.outputPath, "some_file")
}

func TestCreateConfigP521(t *testing.T) {
	cfg, err := CreateConfig(validOPRFP521Ciphersuite, ecgroup.GroupCurve{}, 1, "some_file", -1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, cfg.ocli.Ciphersuite().POG().Name(), "P-521")
	assert.Equal(t, cfg.n, 1)
	assert.Equal(t, cfg.addr, "http://localhost:3001")
	assert.Equal(t, cfg.outputPath, "some_file")
}

func TestInvalidCiphersuite(t *testing.T) {
	_, err := CreateConfig("OPRF-P256-HKDF-SHA512-SSWU-RO", ecgroup.GroupCurve{}, 1, "", -1)
	if err != oerr.ErrUnsupportedGroup {
		t.Fatal("bad group should have triggered a bad ciphersuite error")
	}
}

func TestCreateOPRFRequest(t *testing.T) {
	cfg, err := CreateConfig(validOPRFP384Ciphersuite, ecgroup.GroupCurve{}, 1, "", -1)
	if err != nil {
		t.Fatal(err)
	}
	jsonrpcReq, err := cfg.createOPRFRequest()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, jsonrpcReq.Version, "2.0")
	assert.Equal(t, jsonrpcReq.Method, "eval")
	assert.Equal(t, jsonrpcReq.ID, 1)
	assert.Equal(t, 1, len(jsonrpcReq.Params.Data))
	buf, err := hex.DecodeString(jsonrpcReq.Params.Data[0])
	if err != nil {
		t.Fatal(err)
	}
	// compressedrr encoding
	pog := cfg.ocli.Ciphersuite().POG()
	assert.Equal(t, len(buf), pog.ByteLength()+1)
	ge, err := gg.CreateGroupElement(pog).Deserialize(buf)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, ge.IsValid())
	assert.Equal(t, len(storedBlinds), 1)
	assert.Equal(t, len(storedInputs), 1)

	// check that the point is correctly formed
	geChk, err := pog.EncodeToGroup(storedInputs[0])
	if err != nil {
		t.Fatal(err)
	}
	blindChk, err := geChk.ScalarMult(storedBlinds[0])
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, ge.Equal(blindChk))
}

func TestCreateOPRFRequestBadN(t *testing.T) {
	cfg, err := CreateConfig(validOPRFP384Ciphersuite, ecgroup.GroupCurve{}, -1, "", -1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cfg.createOPRFRequest()
	if err == nil {
		t.Fatal("n < 0 should be unsupported")
	}
}

func TestCreateJSONRPCRequest(t *testing.T) {
	cfg, err := CreateConfig(validOPRFP384Ciphersuite, ecgroup.GroupCurve{}, 1, "", -1)
	if err != nil {
		t.Fatal(err)
	}
	buf0 := []byte{1, 2, 3, 4}
	buf1 := []byte{3, 4, 5, 6}
	jsonrpcReq := cfg.createJSONRPCRequest([][]byte{buf0, buf1}, 3)
	assert.Equal(t, jsonrpcReq.Version, "2.0")
	assert.Equal(t, jsonrpcReq.Method, "eval")
	assert.Equal(t, jsonrpcReq.Params.Data[0], hex.EncodeToString(buf0))
	assert.Equal(t, jsonrpcReq.Params.Data[1], hex.EncodeToString(buf1))
	assert.Equal(t, jsonrpcReq.ID, 3)
}

func TestParseJSONRPCResponseSuccess(t *testing.T) {
	cfg, err := CreateConfig(validOPRFP384Ciphersuite, ecgroup.GroupCurve{}, 1, "", -1)
	if err != nil {
		t.Fatal(err)
	}
	jsonrpcResp := &jsonrpc.ResponseSuccess{
		Version: "2.0",
		Result:  jsonrpc.ResponseResult{Data: []string{"some_response_string"}},
		ID:      1,
	}
	buf, e := json.Marshal(jsonrpcResp)
	if e != nil {
		t.Fatal(e)
	}
	jsonrpcSuccess, err := cfg.parseJSONRPCResponse(buf)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, jsonrpcSuccess.Version, jsonrpcResp.Version)
	assert.Equal(t, jsonrpcSuccess.ID, jsonrpcResp.ID)
	results := jsonrpcSuccess.Result.Data
	assert.Equal(t, len(results), 1)
	assert.Equal(t, results[0], jsonrpcResp.Result.Data[0])
}

func TestParseJSONRPCResponseError(t *testing.T) {
	cfg, err := CreateConfig(validOPRFP384Ciphersuite, ecgroup.GroupCurve{}, 1, "", -1)
	if err != nil {
		t.Fatal(err)
	}
	errorMessage := "error_message"
	errorCode := -33001
	jsonrpcResp := &jsonrpc.ResponseError{
		Version: "2.0",
		Error: oerr.ErrorJSON{
			Message: errorMessage,
			Code:    errorCode,
		},
		ID: 1,
	}
	buf, e := json.Marshal(jsonrpcResp)
	if e != nil {
		t.Fatal(e)
	}
	ret, err := cfg.parseJSONRPCResponse(buf)
	if err == nil {
		fmt.Println(ret)
		t.Fatal("Error should have occurred")
	}
	assert.Equal(t, err, errors.New(errorMessage))
}

func TestParseJSONRPCResponseInvalidResult(t *testing.T) {
	cfg, err := CreateConfig(validOPRFP384Ciphersuite, ecgroup.GroupCurve{}, 1, "", -1)
	if err != nil {
		t.Fatal(err)
	}
	jsonrpcResp := make(map[string]interface{})
	jsonrpcResp["Version"] = "2.0"
	jsonrpcResp["Result"] = 7
	jsonrpcResp["ID"] = 3
	buf, e := json.Marshal(jsonrpcResp)
	if e != nil {
		t.Fatal(e)
	}
	_, err = cfg.parseJSONRPCResponse(buf)
	if err == nil {
		t.Fatal("Server response error should have occurred")
	}
}

func TestParseJSONRPCResponseInvalidField(t *testing.T) {
	cfg, err := CreateConfig(validOPRFP384Ciphersuite, ecgroup.GroupCurve{}, 1, "", -1)
	if err != nil {
		t.Fatal(err)
	}
	jsonrpcResp := make(map[string]interface{})
	jsonrpcResp["Version"] = "2.0"
	jsonrpcResp["Weird"] = []string{"some_string"}
	jsonrpcResp["ID"] = 3
	buf, e := json.Marshal(jsonrpcResp)
	if e != nil {
		t.Fatal(e)
	}
	_, err = cfg.parseJSONRPCResponse(buf)
	if err == nil {
		t.Fatal("Server response error should have occurred")
	}
}
