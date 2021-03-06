package ecgroup

import (
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/alxdavids/voprf-poc/go/oprf/utils"
	"github.com/cloudflare/circl/ecc/p384"
)

type hashToCurveTestVectors struct {
	Vectors []testVector `json:"vectors"`
}

type testVector struct {
	P   expectedPoint `json:"P"`
	Msg string        `json:"msg"`
}

type expectedPoint struct {
	X string `json:"x"`
	Y string `json:"y"`
}

func TestHashToCurveP384(t *testing.T) {
	curve := CreateNistCurve(p384.P384(), sha512.New(), utils.HKDFExtExp{})
	buf, err := ioutil.ReadFile("../../../../test-vectors/hash-to-curve/p384-sha512-sswu-ro-.json")
	if err != nil {
		t.Fatal(err)
	}
	testVectors := hashToCurveTestVectors{}
	err = json.Unmarshal(buf, &testVectors)
	if err != nil {
		t.Fatal(err)
	}
	err = performHashToCurve(curve, testVectors)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHashToCurveP521(t *testing.T) {
	curve := CreateNistCurve(elliptic.P521(), sha512.New(), utils.HKDFExtExp{})
	dir, _ := os.Getwd()
	fmt.Println(dir)
	buf, err := ioutil.ReadFile("../../../../test-vectors/hash-to-curve/p521-sha512-sswu-ro-.json")
	if err != nil {
		t.Fatal(err)
	}
	testVectors := hashToCurveTestVectors{}
	err = json.Unmarshal(buf, &testVectors)
	if err != nil {
		t.Fatal(err)
	}
	err = performHashToCurve(curve, testVectors)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHashToCurve448(t *testing.T) {
	curve := CreateCurve448(sha512.New(), utils.HKDFExtExp{})
	dir, _ := os.Getwd()
	fmt.Println(dir)
	buf, err := ioutil.ReadFile("../../../../test-vectors/hash-to-curve/curve448-sha512-ell2-ro-.json")
	if err != nil {
		t.Fatal(err)
	}
	testVectors := hashToCurveTestVectors{}
	err = json.Unmarshal(buf, &testVectors)
	if err != nil {
		t.Fatal(err)
	}
	err = performHashToCurve(curve, testVectors)
	if err != nil {
		t.Fatal(err)
	}
}

// performHashToCurve performs full hash-to-curve for each of the test inputs
// and checks against expected responses
func performHashToCurve(curve GroupCurve, v hashToCurveTestVectors) error {
	hasher, err := getH2CSuite(curve)
	if err != nil {
		return err
	}
	hasherMod := hasher.(hasher2point)
	hasherMod.dst = []byte("QUUX-V01-CS02")
	for i := range v.Vectors {
		R, err := hasherMod.Hash([]byte(v.Vectors[i].Msg))
		if err != nil {
			return err
		}

		// check point is valid
		if !R.IsValid() {
			return errors.New("Failed to generate a valid point")
		}

		// check test vectors
		// remove prefix
		x := strings.Replace(v.Vectors[i].P.X, "0x", "", 1)
		y := strings.Replace(v.Vectors[i].P.Y, "0x", "", 1)
		expectedX, err := hex.DecodeString(x)
		if err != nil {
			return err
		}
		expectedY, err := hex.DecodeString(y)
		if err != nil {
			return err
		}

		chkR := Point{X: new(big.Int).SetBytes(expectedX), Y: new(big.Int).SetBytes(expectedY), pog: curve, compress: true}
		if !R.Equal(chkR) {
			fmt.Printf("\n expected X in hex %x \n", x)
			fmt.Printf("\n expected Y in hex %x \n", y)
			fmt.Printf("\n X in hex %x \n", hex.EncodeToString(R.X.Bytes()))
			fmt.Printf("\n Y in hex %x \n", hex.EncodeToString(R.Y.Bytes()))
			return errors.New("Points are not equal")
		}
	}
	return nil
}

func BenchmarkHashToCurveP384(b *testing.B) {
	benchmarkHashToCurve(b, CreateNistCurve(p384.P384(), sha512.New(), utils.HKDFExtExp{}))
}

func BenchmarkHashToCurveP521(b *testing.B) {
	benchmarkHashToCurve(b, CreateNistCurve(p384.P384(), sha512.New(), utils.HKDFExtExp{}))
}

func benchmarkHashToCurve(b *testing.B, curve GroupCurve) {
	hasher, err := getH2CSuite(curve)
	if err != nil {
		b.Fatal(err)
	}
	msg := make([]byte, 512)
	_, err = rand.Read(msg)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(msg)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = hasher.Hash(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}
