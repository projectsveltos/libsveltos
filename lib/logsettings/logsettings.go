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

package logsettings

import (
	"context"
	"flag"
	"strconv"
	"sync"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

// Following are log severity levels to be used by sveltos services
const (
	// LogInfo is the info level
	LogInfo = 0
	// LogDebug is the debug level
	LogDebug = 5
	// LogVerbose is an extra level more verbose than Debug
	LogVerbose = 10
)

// LogSetter watches for DebuggingConfiguration and changes log severity at
// run-time based on that custom resource configuration.
type LogSetter struct {
	// Default value. Default to V(0).
	// Use SetDefaultValue to set a different default severity.
	defaultValue string

	logger logr.Logger

	// Setting to severity to Info corresponds to V(0).
	// Use SetInfoValue to set a different severity for info
	infoValue string

	// Setting to severity to Debug corresponds to V(5).
	// Use SetDebugValue to set a different severity for debug
	debugValue string

	// Setting to severity to Verbose corresponds to V(10).
	// Use SetVerboseValue to set a different severity for verbose
	verboseValue string

	// Component registered
	component libsveltosv1beta1.Component

	config *rest.Config
}

var (
	instance *LogSetter
	once     sync.Once
)

func newInstance(component libsveltosv1beta1.Component, config *rest.Config, logger logr.Logger) *LogSetter {
	once.Do(func() {
		logger.Info("Creating LogSetter instance")
		instance = &LogSetter{
			logger:       logger,
			defaultValue: strconv.Itoa(LogInfo),
			infoValue:    strconv.Itoa(LogInfo),
			debugValue:   strconv.Itoa(LogDebug),
			verboseValue: strconv.Itoa(LogVerbose),
			component:    component,
			config:       config,
		}
	})
	return instance
}

// SetDefaultValue sets default severity
func (l *LogSetter) SetDefaultValue(defaultSeverity int) {
	l.defaultValue = strconv.Itoa(defaultSeverity)
}

// SetInfoValue sets severity for Info
func (l *LogSetter) SetInfoValue(infoSeverity int) {
	l.infoValue = strconv.Itoa(infoSeverity)
}

// SetDebugValue sets severity for Debug
func (l *LogSetter) SetDebugValue(debugSeverity int) {
	l.debugValue = strconv.Itoa(debugSeverity)
}

// SetVerboseValue sets severity for Verbose
func (l *LogSetter) SetVerboseValue(verboseSeverity int) {
	l.verboseValue = strconv.Itoa(verboseSeverity)
}

// GetInstance returns LogSetter instance
func GetInstance() *LogSetter {
	return instance
}

// RegisterForLogSettings will react to DebuggingConfiguration change.  Pod
// service account calling this must have permission to read
// DebuggingConfiguration.  DebuggingConfiguration is the custom resource to be
// used to uniformly set log level for all sveltos component.  By calling this
// method, any change in DebuggingConfiguration.Spec will be processed and log
// severity set for affected component(s).
func RegisterForLogSettings(
	ctx context.Context,
	component libsveltosv1beta1.Component,
	logger logr.Logger,
	config *rest.Config,
) *LogSetter {

	logger.Info("Registering for run-time log severity changes", "component", component)
	newInstance(component, config, logger)

	// dynamic informer needs to be told which type to watch
	dcinformer, err := getDynamicInformer(
		"debuggingconfigurations.v1alpha1.lib.projectsveltos.io",
	)
	if err != nil {
		logger.Error(err, "Failed to get informer")
	}
	go runDebuggingConfigurationInformer(ctx.Done(), dcinformer.Informer())
	return instance
}

func getDynamicInformer(resourceType string) (informers.GenericInformer, error) {
	// Grab a dynamic interface that we can create informers from
	dc, err := dynamic.NewForConfig(instance.config)
	if err != nil {
		return nil, err
	}
	// Create a factory object that can generate informers for resource types
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		dc,
		0,
		corev1.NamespaceAll,
		nil,
	)
	gvr, _ := schema.ParseResourceArg(resourceType)
	informer := factory.ForResource(*gvr)
	return informer, nil
}

func runDebuggingConfigurationInformer(
	stopCh <-chan struct{},
	s cache.SharedIndexInformer,
) {

	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			instance.logger.Info("got add notification for DebuggingConfiguration")
			d := &libsveltosv1beta1.DebuggingConfiguration{}
			err := runtime.DefaultUnstructuredConverter.
				FromUnstructured(obj.(*unstructured.Unstructured).UnstructuredContent(), d)
			if err != nil {
				instance.logger.Error(err, "could not convert obj to DebuggingConfiguration")
				return
			}
			UpdateLogLevel(d)
		},
		DeleteFunc: func(obj interface{}) {
			instance.logger.Info(
				"DebuggingConfiguration is deleted. Setting log severity to info",
				"default",
				instance.defaultValue,
			)
			if err := flag.Lookup("v").Value.Set(instance.defaultValue); err != nil {
				instance.logger.Error(err, "unable to set default level")
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			instance.logger.Info("got update notification for DebuggingConfiguration")
			d := &libsveltosv1beta1.DebuggingConfiguration{}
			err := runtime.DefaultUnstructuredConverter.
				FromUnstructured(newObj.(*unstructured.Unstructured).UnstructuredContent(), d)
			if err != nil {
				instance.logger.Error(err, "could not convert obj to DebuggingConfiguration")
				return
			}
			UpdateLogLevel(d)
		},
	}
	_, err := s.AddEventHandler(handlers)
	if err != nil {
		panic(1)
	}
	s.Run(stopCh)
}

// UpdateLogLevel updates log severity
func UpdateLogLevel(
	d *libsveltosv1beta1.DebuggingConfiguration,
) {

	found := false
	for _, c := range d.Spec.Configuration {
		if instance.component == c.Component {
			if c.LogLevel == libsveltosv1beta1.LogLevelVerbose {
				found = true
				instance.logger.Info("Setting log severity to verbose", "verbose", instance.verboseValue)
				if err := flag.Lookup("v").Value.Set(instance.verboseValue); err != nil {
					instance.logger.Error(err, "unable to set log level")
				}
			} else if c.LogLevel == libsveltosv1beta1.LogLevelDebug {
				found = true
				instance.logger.Info("Setting log severity to debug", "debug", instance.debugValue)
				if err := flag.Lookup("v").Value.Set(instance.debugValue); err != nil {
					instance.logger.Error(err, "unable to set log level")
				}
			} else if c.LogLevel == libsveltosv1beta1.LogLevelInfo {
				found = true
				instance.logger.Info("Setting log severity to info", "info", instance.infoValue)
				if err := flag.Lookup("v").Value.Set(instance.infoValue); err != nil {
					instance.logger.Error(err, "unable to set log level")
				}
			}
		}
	}

	if !found {
		instance.logger.Info("Setting log severity to info", "default", instance.defaultValue)
		if err := flag.Lookup("v").Value.Set(instance.defaultValue); err != nil {
			instance.logger.Error(err, "unable to set default level")
		}
	}
}
