package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/akeyless-community/akeyless-cert-manager-issuer/api/v1alpha1"
	"github.com/akeyless-community/akeyless-cert-manager-issuer/controller"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var version = "v0.0.0-dev"

const inClusterNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

func main() {
	opts := ctrlzap.Options{}
	opts.BindFlags(flag.CommandLine)

	var metricsAddr, probeAddr string
	var enableLeaderElection bool
	var maxRetryDuration time.Duration
	var clusterResourceNamespace string
	var leaderElectionNamespace string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metrics endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address for health probes.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true, "Enable leader election.")
	flag.StringVar(&leaderElectionNamespace, "leader-election-namespace", "", "Namespace for Lease objects when running outside the cluster (required if --leader-elect=true locally). Ignored in-cluster.")
	flag.DurationVar(&maxRetryDuration, "max-retry-duration", 2*time.Minute, "Max time to retry signing after CertificateRequest creation.")
	flag.StringVar(&clusterResourceNamespace, "cluster-resource-namespace", "", "Namespace where secrets for AkeylessClusterIssuer are stored (e.g. cert-manager).")
	flag.Parse()

	opts.StacktraceLevel = zapcore.DPanicLevel
	log := ctrlzap.New(ctrlzap.UseFlagOptions(&opts)).WithName("akeyless-cert-manager-issuer")

	if atomlvl, ok := opts.Level.(zap.AtomicLevel); ok {
		zaplvl := atomlvl.Level()
		kloglvl := 0
		if zaplvl < 0 {
			kloglvl = -int(zaplvl)
		}
		dummy := flag.FlagSet{}
		klog.InitFlags(&dummy)
		_ = dummy.Set("v", strconv.Itoa(kloglvl))
		_ = dummy.Parse(nil)
	}
	klog.SetLogger(log)
	ctrl.SetLogger(log)

	if err := run(clusterResourceNamespace, metricsAddr, enableLeaderElection, leaderElectionNamespace, probeAddr, maxRetryDuration); err != nil {
		log.Error(err, "fatal error")
		os.Exit(1)
	}
}

func run(clusterResourceNamespace, metricsAddr string, enableLeaderElection bool, leaderElectionNamespace, probeAddr string, maxRetry time.Duration) error {
	setupLog := ctrl.Log.WithName("setup")
	setupLog.Info("starting controller", "version", version)

	inCluster := runningInCluster()
	if err := getInClusterNamespace(&clusterResourceNamespace); err != nil {
		if errors.Is(err, errNotInCluster) {
			setupLog.Info("not in cluster: use --cluster-resource-namespace for AkeylessClusterIssuer support")
		} else {
			return fmt.Errorf("in-cluster namespace: %w", err)
		}
	}

	if enableLeaderElection && !inCluster && leaderElectionNamespace == "" {
		return fmt.Errorf("outside the cluster with --leader-elect=true: set --leader-election-namespace (e.g. akeyless-cert-test) so controller-runtime can create Leases, or use --leader-elect=false for local dev")
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: 9443,
		}),
		HealthProbeBindAddress:        probeAddr,
		LeaderElection:                enableLeaderElection,
		LeaderElectionID:              "fd4a8c21.certmanager.akeyless.io",
		LeaderElectionNamespace:       leaderElectionNamespace,
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		return fmt.Errorf("new manager: %w", err)
	}

	sgn := &controller.Signer{ClusterResourceNamespace: clusterResourceNamespace}
	ctx := ctrl.SetupSignalHandler()
	if err := sgn.SetupWithManager(ctx, mgr, maxRetry); err != nil {
		return fmt.Errorf("setup controller: %w", err)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("healthz: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("readyz: %w", err)
	}

	setupLog.Info("controller running")
	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("manager: %w", err)
	}
	return nil
}

var errNotInCluster = errors.New("not running in-cluster")

func runningInCluster() bool {
	_, err := os.Stat(inClusterNamespacePath)
	return err == nil
}

func getInClusterNamespace(ns *string) error {
	if *ns != "" {
		return nil
	}
	_, err := os.Stat(inClusterNamespacePath)
	if os.IsNotExist(err) {
		return errNotInCluster
	}
	if err != nil {
		return err
	}
	b, err := os.ReadFile(inClusterNamespacePath)
	if err != nil {
		return err
	}
	*ns = string(b)
	return nil
}
