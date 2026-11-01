/*
Copyright 2026. projectsveltos.io. All rights reserved.

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
	"sync"

	"k8s.io/client-go/rest"
)

var (
	restConfigLimitsMtx sync.RWMutex
	restConfigQPS       float32
	restConfigBurst     int
)

// SetRestConfigLimits sets the QPS and burst applied to every rest.Config returned for a
// managed cluster. A value of zero keeps the client-go default (5 QPS, burst 10).
// It is meant to be called once at startup, before any rest.Config is requested. A rest.Config
// that was already returned, or is held in a cache, keeps the limits it was created with.
func SetRestConfigLimits(qps float32, burst int) {
	restConfigLimitsMtx.Lock()
	defer restConfigLimitsMtx.Unlock()

	restConfigQPS = qps
	restConfigBurst = burst
}

// applyRestConfigLimits sets the configured QPS and burst on config.
// It must only be called on a rest.Config not yet shared, as it writes to it.
func applyRestConfigLimits(config *rest.Config) {
	restConfigLimitsMtx.RLock()
	defer restConfigLimitsMtx.RUnlock()

	config.QPS = restConfigQPS
	config.Burst = restConfigBurst
}
