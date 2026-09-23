package fingerprint

import "testing"

func TestEmbeddedCatalogContainsAllPyMaxVersions(t *testing.T) {
	catalog := NewVersionCatalog()
	versions := catalog.Versions()
	if len(versions) < 40 {
		t.Fatalf("embedded catalog is incomplete: got %d versions", len(versions))
	}
	fingerprint, err := catalog.Resolve(RecommendedAppVersion)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprint.VersionCode != 6790 || len(fingerprint.SoMetaSha256) != 4 {
		t.Fatalf("unexpected recommended fingerprint: %#v", fingerprint)
	}
}
