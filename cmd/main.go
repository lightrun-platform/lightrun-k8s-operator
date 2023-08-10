/*
Copyright 2022 Lightrun

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

package main

import (
	"errors"
	"flag"
	"os"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"go.uber.org/zap/zapcore"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	zaplog "go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	agentsv1beta "github.com/lightrun-platform/lightrun-k8s-operator/api/v1beta"
	"github.com/lightrun-platform/lightrun-k8s-operator/internal/controller"
	//+kubebuilder:scaffold:imports
)

var (
	scheme               = runtime.NewScheme()
	setupLog             = ctrl.Log.WithName("setup")
	watchNamespaceEnvVar = "WATCH_NAMESPACE"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(agentsv1beta.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func getWatchNamespaces() ([]string, error) {
	var namespaces []string
	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found || (found && ns == "") {
		return nil, errors.New(watchNamespaceEnvVar + " Env Var not set or empty")
	}
	nsSlice := strings.Split(ns, ",")
	for _, ns := range nsSlice {
		if ns != "" {
			namespaces = append(namespaces, ns)
		}
	}
	return namespaces, nil
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	opts := zap.Options{
		Development:     false,
		ZapOpts:         []zaplog.Option{zaplog.AddCaller(), zaplog.AddCallerSkip(-1)},
		StacktraceLevel: zapcore.DPanicLevel,
		TimeEncoder:     zapcore.ISO8601TimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	setupLog.Info("Log verbosity", "V", opts.Level)

	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "5b425f09.lightrun.com",

		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	}

	watchNamespaces, err := getWatchNamespaces()
	if err != nil {
		setupLog.Info("Controller will watch and manage resources in all namespaces")
	} else {
		setupLog.Info("Controller will watch following namespaces", "namespaces", watchNamespaces)
		options.Cache.Namespaces = watchNamespaces
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controller.LightrunJavaAgentReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("LightrunJavaAgent"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LightrunJavaAgent")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
