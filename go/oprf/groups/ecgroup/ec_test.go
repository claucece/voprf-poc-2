package ecgroup

import (
	"crypto/elliptic"
	"crypto/sha512"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/alxdavids/voprf-poc/go/oerr"
	gg "github.com/alxdavids/voprf-poc/go/oprf/groups"
	"github.com/alxdavids/voprf-poc/go/oprf/utils"
	"github.com/alxdavids/voprf-poc/go/oprf/utils/constants"
	"github.com/cloudflare/circl/ecc/p384"
	"github.com/stretchr/testify/assert"
)

var (
	testCurves = []string{"P-384", "P-521", "curve-448"}
)

func TestCiphersuiteFromString(t *testing.T) {
	for _, b := range []bool{false, true} {
		for _, c := range testCurves {
			ciphersuiteFromString(t, c, b)
		}
	}
}

func TestCiphersuiteFromStringInvalidH2C(t *testing.T) {
	for _, c := range testCurves {
		ciphersuiteFromStringInvalidH2C(t, c)
	}
}

func TestCiphersuiteFromStringInvalidHash(t *testing.T) {
	for _, c := range testCurves {
		ciphersuiteFromStringInvalidHash(t, c)
	}
}

func TestCiphersuiteFromStringInvalidGroup(t *testing.T) {
	ciphersuiteFromStringInvalidGroup(t, "P-256")
}

func TestGroupCurveEncodingP384(t *testing.T) {
	curve := CreateNistCurve(p384.P384(), sha512.New(), utils.HKDFExtExp{})
	_, err := curveEncoding(curve)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGroupCurvePointSerializationP384(t *testing.T) {
	curve := CreateNistCurve(p384.P384(), sha512.New(), utils.HKDFExtExp{})
	P, err := curveEncoding(curve)
	if err != nil {
		t.Fatal(err)
	}

	checkSerialize(curve, P)
}

func TestGroupCurvePointSerializationWithCompressionP384(t *testing.T) {
	curve := CreateNistCurve(p384.P384(), sha512.New(), utils.HKDFExtExp{})
	P, err := curveEncoding(curve)
	if err != nil {
		t.Fatal(err)
	}

	P.compress = true
	checkSerialize(curve, P)
}

func TestGroupCurveEncodingP521(t *testing.T) {
	curve := CreateNistCurve(elliptic.P521(), sha512.New(), utils.HKDFExtExp{})
	_, err := curveEncoding(curve)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGroupCurvePointSerializationP521(t *testing.T) {
	curve := CreateNistCurve(elliptic.P521(), sha512.New(), utils.HKDFExtExp{})
	P, err := curveEncoding(curve)
	if err != nil {
		t.Fatal(err)
	}

	checkSerialize(curve, P)
}

func TestGroupCurvePointSerializationWithCompressionP521(t *testing.T) {
	curve := CreateNistCurve(elliptic.P521(), sha512.New(), utils.HKDFExtExp{})
	P, err := curveEncoding(curve)
	if err != nil {
		t.Fatal(err)
	}

	P.compress = true
	checkSerialize(curve, P)
}

func TestPointAddition(t *testing.T) {
	curve := CreateNistCurve(elliptic.P521(), sha512.New(), utils.HKDFExtExp{})
	P, err := curveEncoding(curve)
	if err != nil {
		t.Fatal(err)
	}
	P1, err := curveEncoding(curve)
	if err != nil {
		t.Fatal(err)
	}
	P2, err := curveEncoding(curve)
	if err != nil {
		t.Fatal(err)
	}
	x, err := curve.UniformFieldElement()
	if err != nil {
		t.Fatal(err)
	}
	y, err := curve.UniformFieldElement()
	if err != nil {
		t.Fatal(err)
	}
	xP1, err := P1.ScalarMult(x)
	if err != nil {
		t.Fatal(err)
	}
	yP2, err := P2.ScalarMult(y)
	if err != nil {
		t.Fatal(err)
	}
	xP1yP2, err := xP1.Add(yP2)
	if err != nil {
		t.Fatal(err)
	}
	xyP, err := P.ScalarMult(new(big.Int).Add(x, y))
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, xyP.Equal(xP1yP2))
}

func TestPointEqualityP384(t *testing.T) {
	checkPointEquality(t, p384.P384())
}

func TestPointEqualityP521(t *testing.T) {
	checkPointEquality(t, elliptic.P521())
}

func TestPointEqualityFailsOnBadGroups(t *testing.T) {
	p384 := CreateNistCurve(p384.P384(), sha512.New(), utils.HKDFExtExp{})
	P, err := curveEncoding(p384)
	if err != nil {
		t.Fatal(err)
	}
	p521 := CreateNistCurve(elliptic.P521(), sha512.New(), utils.HKDFExtExp{})
	Q, err := curveEncoding(p521)
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, P.Equal(Q))
}

func TestPointEqualityFailsOnInvalidCallerPoint(t *testing.T) {
	p384 := CreateNistCurve(p384.P384(), sha512.New(), utils.HKDFExtExp{})
	P, err := curveEncoding(p384)
	if err != nil {
		t.Fatal(err)
	}
	P.X = constants.MinusOne
	p521 := CreateNistCurve(elliptic.P521(), sha512.New(), utils.HKDFExtExp{})
	Q, err := curveEncoding(p521)
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, P.Equal(Q))
}

func TestPointEqualityFailsOnInvalidInputPoint(t *testing.T) {
	p384 := CreateNistCurve(p384.P384(), sha512.New(), utils.HKDFExtExp{})
	P, err := curveEncoding(p384)
	if err != nil {
		t.Fatal(err)
	}
	p521 := CreateNistCurve(elliptic.P521(), sha512.New(), utils.HKDFExtExp{})
	Q, err := curveEncoding(p521)
	if err != nil {
		t.Fatal(err)
	}
	Q.X = constants.MinusOne
	assert.False(t, P.Equal(Q))
}

func checkPointEquality(t *testing.T, curve elliptic.Curve) {
	nistCurve := CreateNistCurve(curve, sha512.New(), utils.HKDFExtExp{})
	P, err := curveEncoding(nistCurve)
	if err != nil {
		t.Fatal(err)
	}
	Q := P
	assert.True(t, P.Equal(Q))
}

func ciphersuiteFromString(t *testing.T, groupName string, verifiable bool) {
	var s, ciphName string

	if verifiable {
		s = "V"
	}

	if groupName == "P-384" || groupName == "P-521" {
		ciphName = fmt.Sprintf("%sOPRF-%s-HKDF-SHA512-SSWU-RO", s, strings.ReplaceAll(groupName, "-", ""))
	} else if groupName == "curve-448" {
		ciphName = fmt.Sprintf("%sOPRF-%s-HKDF-SHA512-ELL2-RO", s, strings.ReplaceAll(groupName, "-", ""))
	}

	ciph, err := gg.Ciphersuite{}.FromString(ciphName, GroupCurve{})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, ciph.Name(), ciphName)
	assert.Equal(t, ciph.H3(), sha512.New())
	assert.Equal(t, ciph.H4(), sha512.New())
	assert.Equal(t, ciph.POG().Name(), groupName)
	assert.Equal(t, ciph.Verifiable(), verifiable)
	if verifiable {
		assert.Equal(t, reflect.TypeOf(ciph.H5()).Name(), "HKDFExtExp")
	} else {
		assert.Equal(t, ciph.H5(), nil)
	}
}

func ciphersuiteFromStringInvalidH2C(t *testing.T, groupName string) {
	ciphName := fmt.Sprintf("OPRF-%s-HKDF-SHA512-ELL1", strings.ReplaceAll(groupName, "-", ""))
	_, err := gg.Ciphersuite{}.FromString(ciphName, GroupCurve{})
	if err != oerr.ErrUnsupportedH2C {
		t.Fatal("Error didn't occur")
	}
}

func ciphersuiteFromStringInvalidHash(t *testing.T, groupName string) {
	ciphName := fmt.Sprintf("OPRF-%s-HKDF-SHA256-SSWU-RO", strings.ReplaceAll(groupName, "-", ""))
	_, err := gg.Ciphersuite{}.FromString(ciphName, GroupCurve{})
	if err != oerr.ErrUnsupportedHash {
		t.Fatal("Error didn't occur")
	}
}

func ciphersuiteFromStringInvalidGroup(t *testing.T, groupName string) {
	ciphName := fmt.Sprintf("OPRF-%s-HKDF-SHA512-SSWU-RO", strings.ReplaceAll(groupName, "-", ""))
	_, err := gg.Ciphersuite{}.FromString(ciphName, GroupCurve{})
	if err != oerr.ErrUnsupportedGroup {
		t.Fatal("Error didn't occur")
	}
}

func curveEncoding(curve GroupCurve) (Point, error) {
	P, err := curve.EncodeToGroup([]byte("test"))
	if err != nil {
		return Point{}, err
	}

	if !P.IsValid() {
		return Point{}, errors.New("Didn't generated valid curve point")
	}

	ret, err := castToPoint(P)
	if err != nil {
		return Point{}, err
	}

	return ret, nil
}

func checkSerialize(curve GroupCurve, P Point) error {
	buf, err := P.Serialize()
	if err != nil {
		return err
	}
	Q, err := Point{}.New(curve).Deserialize(buf)
	if err != nil {
		return err
	}

	if (P.compress && len(buf) != curve.ByteLength()+1) ||
		len(buf) != (2*curve.ByteLength())+1 {
		return errors.New("Incorrect buffer length")
	}

	qPoint, err := castToPoint(Q)
	if err != nil {
		return err
	}
	if qPoint.Equal(P) {
		return errors.New("qPoint and P are not equal points")
	}
	return nil
}
