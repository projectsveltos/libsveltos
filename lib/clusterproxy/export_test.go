/*
Copyright 2022. projectsveltos.io. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clusterproxy

import (
	"time"

	"k8s.io/client-go/rest"
)

const (
	CapiKubeconfigSecretNamePostfix = capiKubeconfigSecretNamePostfix

	SveltosKubeconfigSecretNamePostfix = sveltosKubeconfigSecretNamePostfix
)

var (
	IsSveltosClusterInPullMode = isSveltosClusterInPullMode
)

// StoreTestWiCache inserts a pre-built entry into the workload identity cache.
// For use in tests only.
func StoreTestWiCache(namespace, name string, cfg *rest.Config, expiresAt time.Time) {
	wiCache.Store(wiCacheKey(namespace, name), cachedRestConfig{config: cfg, expiresAt: expiresAt})
}

// LoadTestWiCache returns the cached rest.Config and expiry for the given cluster.
// For use in tests only.
func LoadTestWiCache(namespace, name string) (*rest.Config, time.Time, bool) {
	v, ok := wiCache.Load(wiCacheKey(namespace, name))
	if !ok {
		return nil, time.Time{}, false
	}
	entry := v.(cachedRestConfig)
	return entry.config, entry.expiresAt, true
}

// GetCADataForTest calls the internal getCAData function.
// For use in tests only.
var GetCADataForTest = getCAData
