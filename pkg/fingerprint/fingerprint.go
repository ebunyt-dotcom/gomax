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

// DefaultFingerprint returns a default Android mobile build fingerprint.
func DefaultFingerprint() *ApkBuildFingerprint {
	return &ApkBuildFingerprint{
		VersionCode:           24020,
		VersionName:           "24.2.0",
		CertificateMetaSha256: "b0769b76c8bbdb0407137456d2524d77519a4a7cfcb7863bf68a18fa3080ffb0",
		DexMetaSha256:         "3c0f4f7d45f4ea6c45c116d47b5ef6f279f64e229f3d9b4b9b99df8996bdfdf8",
		SoMetaSha256: map[string]string{
			"arm64-v8a": "5a2f5f7dc5f4ea6c45c116d47b5ef6f279f64e229f3d9b4b9b99df8996bdfd01",
			"armeabi-v7a": "5a2f5f7dc5f4ea6c45c116d47b5ef6f279f64e229f3d9b4b9b99df8996bdfd02",
			"x86_64":    "5a2f5f7dc5f4ea6c45c116d47b5ef6f279f64e229f3d9b4b9b99df8996bdfd03",
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
