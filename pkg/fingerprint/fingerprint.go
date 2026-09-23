package fingerprint

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
)

const DefaultRemoteCatalogURL = "https://hashes.pymax.org/versions.json"
const RecommendedAppVersion = "26.25.0"

//go:embed apk_fingerprints.json
var embeddedCatalog []byte

type VersionCatalog struct {
	mu       sync.RWMutex
	versions map[string]*ApkBuildFingerprint
}

func NewVersionCatalog() *VersionCatalog {
	catalog := &VersionCatalog{versions: make(map[string]*ApkBuildFingerprint)}
	var raw map[string]struct {
		Certificate string            `json:"certificate_meta_sha256"`
		Dex         string            `json:"dex_meta_sha256"`
		SO          map[string]string `json:"so_meta_sha256"`
		Build       int               `json:"build_number"`
	}
	if json.Unmarshal(embeddedCatalog, &raw) == nil {
		for version, item := range raw {
			catalog.versions[version] = &ApkBuildFingerprint{VersionName: version, VersionCode: item.Build, CertificateMetaSha256: item.Certificate, DexMetaSha256: item.Dex, SoMetaSha256: item.SO}
		}
	}
	if len(catalog.versions) == 0 {
		fallback := hardcodedDefaultFingerprint()
		catalog.versions[fallback.VersionName] = fallback
	}
	return catalog
}

// Versions returns all known built-in and remotely loaded versions in order.
func (c *VersionCatalog) Versions() []string {
	c.mu.RLock()
	versions := make([]string, 0, len(c.versions))
	for version := range c.versions {
		versions = append(versions, version)
	}
	c.mu.RUnlock()
	sort.Strings(versions)
	return versions
}

func (c *VersionCatalog) Add(version string, value *ApkBuildFingerprint, override bool) error {
	if version == "" || value == nil {
		return errors.New("fingerprint: version and value are required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.versions[version]; exists && !override {
		return fmt.Errorf("fingerprint: version %s already exists", version)
	}
	copy := *value
	c.versions[version] = &copy
	return nil
}

func (c *VersionCatalog) Resolve(version string) (*ApkBuildFingerprint, error) {
	c.mu.RLock()
	value, ok := c.versions[version]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("fingerprint: version %s not found", version)
	}
	copy := *value
	return &copy, nil
}

func (c *VersionCatalog) LoadRemote(ctx context.Context, client *http.Client, catalogURL string) error {
	if client == nil {
		client = http.DefaultClient
	}
	if catalogURL == "" {
		catalogURL = DefaultRemoteCatalogURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, catalogURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("fingerprint: catalog HTTP status %s", resp.Status)
	}
	var raw map[string]struct {
		Certificate string            `json:"certificate_meta_sha256"`
		Dex         string            `json:"dex_meta_sha256"`
		SO          map[string]string `json:"so_meta_sha256"`
		Build       int               `json:"build_number"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return err
	}
	for version, item := range raw {
		_ = c.Add(version, &ApkBuildFingerprint{VersionName: version, VersionCode: item.Build, CertificateMetaSha256: item.Certificate, DexMetaSha256: item.Dex, SoMetaSha256: item.SO}, true)
	}
	return nil
}

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

func (g *FingerprintGenerator) VersionName() string {
	if g == nil || g.data == nil {
		return ""
	}
	return g.data.VersionName
}

// DefaultFingerprint returns a default Android mobile build fingerprint matching PyMax 26.25.0 (build 6790).
func DefaultFingerprint() *ApkBuildFingerprint {
	if resolved, err := NewVersionCatalog().Resolve(RecommendedAppVersion); err == nil {
		return resolved
	}
	return hardcodedDefaultFingerprint()
}

func hardcodedDefaultFingerprint() *ApkBuildFingerprint {
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
	if g == nil || g.data == nil {
		return nil, errors.New("fingerprint data is nil")
	}
	if arch == "" {
		arch = "arm64-v8a"
	}

	soHashHex, ok := g.data.SoMetaSha256[arch]
	if !ok {
		return nil, fmt.Errorf("fingerprint: unsupported architecture %q", arch)
	}
	if soHashHex == "" {
		return nil, errors.New("no SO meta sha256 found for architecture")
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
