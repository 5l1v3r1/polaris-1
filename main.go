// Copyright 2018 ReactiveOps
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"os"

	conf "github.com/reactiveops/fairwinds/pkg/config"
	"github.com/reactiveops/fairwinds/pkg/validator"
	"github.com/spf13/viper"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
)

// FairwindsName is used for Kubernetes resource naming
var FairwindsName = "fairwinds"
var log = logf.Log.WithName(FairwindsName)

func main() {
	var disableWebhookConfigInstaller bool
	flag.BoolVar(&disableWebhookConfigInstaller, "disable-webhook-config-installer", false,
		"disable the installer in the webhook server, so it won't install webhook configuration resources during bootstrapping")

	flag.Parse()
	logf.SetLogger(logf.ZapLogger(false))
	entryLog := log.WithName("entrypoint")

	// Parse config.
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	var conf conf.Configuration
	if err := viper.ReadInConfig(); err != nil {
		entryLog.Error(err, "config err")
	}

	if err := viper.Unmarshal(&conf); err != nil {
		entryLog.Error(err, "unmarshal config err")
	}

	entryLog.Info(fmt.Sprintf("conf requests: %s", conf.Requests))
	entryLog.Info(fmt.Sprintf("conf limits: %s", conf.Limits))

	// Setup a Manager
	entryLog.Info("setting up manager")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		entryLog.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	podValidatingWebhook, err := builder.NewWebhookBuilder().
		Name("validating.k8s.io").
		Validating().
		Operations(admissionregistrationv1beta1.Create, admissionregistrationv1beta1.Update).
		WithManager(mgr).
		ForType(&corev1.Pod{}).
		Handlers(&validator.PodValidator{}).
		Build()
	if err != nil {
		entryLog.Error(err, "unable to setup validating webhook")
		os.Exit(1)
	}

	entryLog.Info("setting up webhook server")
	as, err := webhook.NewServer(FairwindsName, mgr, webhook.ServerOptions{
		Port:                          9876,
		CertDir:                       "/tmp/cert",
		DisableWebhookConfigInstaller: &disableWebhookConfigInstaller,
		BootstrapOptions: &webhook.BootstrapOptions{
			ValidatingWebhookConfigName: FairwindsName,
			Secret: &apitypes.NamespacedName{
				Namespace: FairwindsName,
				Name:      FairwindsName,
			},

			Service: &webhook.Service{
				Namespace: FairwindsName,
				Name:      FairwindsName,

				// Selectors should select the pods that runs this webhook server.
				Selectors: map[string]string{
					"app": FairwindsName,
				},
			},
		},
	})
	if err != nil {
		entryLog.Error(err, "unable to create a new webhook server")
		os.Exit(1)
	}

	entryLog.Info("registering webhooks to the webhook server")
	if err = as.Register(podValidatingWebhook); err != nil {
		entryLog.Error(err, "unable to register webhooks in the admission server")
		os.Exit(1)
	}

	entryLog.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		entryLog.Error(err, "unable to run manager")
		os.Exit(1)
	}
}
