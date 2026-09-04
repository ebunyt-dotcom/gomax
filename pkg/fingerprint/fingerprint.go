package fingerprint

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
)

// ApkBuildFingerprint holds the cryptographic hash metadata of the client build.
type ApkBuildFingerprint struct {
	VersionCode           int               `json:"version_code"`
	VersionName           string            `json:"version_name"`
	CertificateMetaSha256 string            `json:"certificate_meta_sha256"`
	DexMetaSha256         string            `json:"dex_meta_sha256"`
	SoMetaSha256          map[string]string `json:"so_meta_sha256"`
}

// FingerprintGenerator reproduces the exact hardware/APK security fingerprint required by Max API.
type FingerprintGenerator struct {
	data *ApkBuildFingerprint
}

// NewFingerprintGenerator creates a new generator with the specified build metadata.
func NewFingerprintGenerator(data *ApkBuildFingerprint) *FingerprintGenerator {
	return &FingerprintGenerator{data: data}
}

// DefaultFingerprint returns a default Android mobile build fingerprint matching PyMax 26.25.0 (build 6790).
func DefaultFingerprint() *ApkBuildFingerprint {
	return &ApkBuildFingerprint{
		VersionCode:           6790,
		VersionName:           "26.25.0",
		CertificateMetaSha256: "1684414033eb263e2c615f8b7df5ed8793850a07656304997fbf07e9e21e1e93",
		DexMetaSha256:         "8db68fcc0e85e8f041fe4a875c0a9bcfe542a8f679603728c651ed81b64dd684",
		SoMetaSha256: map[string]string{
			"arm64-v8a":   "634ecc42b246784d975f180b4fecf903df235cdf0476da47163a85630eb1a6a8",
			"armeabi-v7a": "042220bdd481a280d2c1f4f6827f0e4fab7bca61e5af0f6035a0d191aed1350c",
			"x86":         "deffe34d2a9d83584e02cbb3f22ba5a6dbe1b065dbc8a8ea8ca908dae865c5f6",
			"x86_64":      "251b88c27a1c055f27adc110e44a75a1c60408b0d5e20e3844f816aa227212a3",
		},
	}
}

// GenerateFingerprint calculates the 96-byte combined SHA256 signature (Cert + Dex + Lib).
func (g *FingerprintGenerator) GenerateFingerprint(deviceID string, callsSeed int64, arch string) ([]byte, error) {
	if g.data == nil {
		return nil, errors.New("fingerprint data is nil")
	}
	if arch == "" {
		arch = "arm64-v8a"
	}

	soHashHex, ok := g.data.SoMetaSha256[arch]
	if !ok {
		// Fallback to first available arch
		for _, v := range g.data.SoMetaSha256 {
			soHashHex = v
			break
		}
	}

	certBytes, err := hex.DecodeString(g.data.CertificateMetaSha256)
	if err != nil {
		return nil, err
	}
	dexBytes, err := hex.DecodeString(g.data.DexMetaSha256)
	if err != nil {
		return nil, err
	}
	soBytes, err := hex.DecodeString(soHashHex)
	if err != nil {
		return nil, err
	}

	seedBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(seedBytes, uint64(callsSeed))
	deviceBytes := []byte(deviceID)

	// H1 = SHA256(Cert + Seed + DeviceID)
	h1 := sha256.New()
	h1.Write(certBytes)
	h1.Write(seedBytes)
	h1.Write(deviceBytes)
	digest1 := h1.Sum(nil)

	// H2 = SHA256(Dex + Seed + DeviceID)
	h2 := sha256.New()
	h2.Write(dexBytes)
	h2.Write(seedBytes)
	h2.Write(deviceBytes)
	digest2 := h2.Sum(nil)

	// H3 = SHA256(So + Seed + DeviceID)
	h3 := sha256.New()
	h3.Write(soBytes)
	h3.Write(seedBytes)
	h3.Write(deviceBytes)
	digest3 := h3.Sum(nil)

	result := make([]byte, 0, 96)
	result = append(result, digest1...)
	result = append(result, digest2...)
	result = append(result, digest3...)
	return result, nil
}
