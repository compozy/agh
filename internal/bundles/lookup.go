package bundles

import (
	"strings"

	extensionpkg "github.com/pedronauck/agh/internal/extension"
	"github.com/pedronauck/agh/internal/resources"
)

type bundleRecordKey struct {
	extensionName string
	bundleName    string
}

type bundleRecordLookup struct {
	exact   map[bundleRecordKey]int
	records []resources.Record[BundleResourceSpec]
}

func newBundleRecordLookup(records []resources.Record[BundleResourceSpec]) bundleRecordLookup {
	exact := make(map[bundleRecordKey]int, len(records))
	for idx, record := range records {
		key := newBundleRecordKey(record.Spec.ExtensionName, record.Spec.Bundle.Name)
		if key.extensionName == "" || key.bundleName == "" {
			continue
		}
		exact[key] = idx
	}
	return bundleRecordLookup{
		exact:   exact,
		records: records,
	}
}

func findBundleResourceRecordIndexed(
	lookup bundleRecordLookup,
	extensionName string,
	bundleName string,
) (resources.Record[BundleResourceSpec], bool) {
	key := newBundleRecordKey(extensionName, bundleName)
	idx, ok := lookup.exact[key]
	if ok {
		return lookup.records[idx], true
	}
	for _, candidate := range lookup.records {
		if strings.EqualFold(strings.TrimSpace(candidate.Spec.ExtensionName), key.extensionName) &&
			strings.EqualFold(strings.TrimSpace(candidate.Spec.Bundle.Name), key.bundleName) {
			return candidate, true
		}
	}
	return resources.Record[BundleResourceSpec]{}, false
}

func newBundleRecordKey(extensionName string, bundleName string) bundleRecordKey {
	return bundleRecordKey{
		extensionName: strings.ToLower(strings.TrimSpace(extensionName)),
		bundleName:    strings.ToLower(strings.TrimSpace(bundleName)),
	}
}

func findProfile(items []extensionpkg.BundleProfile, name string) (extensionpkg.BundleProfile, bool) {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Name), strings.TrimSpace(name)) {
			return item, true
		}
	}
	return extensionpkg.BundleProfile{}, false
}
