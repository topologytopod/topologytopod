package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	runtimeWebhook "sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	debug       bool
	webhookPort int
)

func main() {
	flag.IntVar(&webhookPort, "port", 443, "webhook server port")
	flag.BoolVar(&debug, "debug", true, "debug mode")
	flag.Parse()
	log.SetLogger(zap.New(zap.UseDevMode(false)))

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		fmt.Printf("faild to add clientgoscheme to scheme: %v", err)
		os.Exit(1)
	}
	cli, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		fmt.Printf("")
		os.Exit(1)
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		WebhookServer: runtimeWebhook.NewServer(runtimeWebhook.Options{
			Port:    webhookPort,
			CertDir: "/etc/webhook/certs",
		}),
	})
	if err != nil {
		fmt.Printf("faild to new manager: %v", err)
		os.Exit(1)
	}
	m := mutate{cli: cli}
	mgr.GetWebhookServer().Register("/mutate", &webhook.Admission{
		Handler: admission.HandlerFunc(m.mutateHookPod),
	})
	ctrl.Log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		fmt.Printf("faild to start manager: %v", err)
		os.Exit(1)
	}
}

type mutate struct {
	cli client.Client
}

func (m *mutate) mutateHookPod(ctx context.Context, req admission.Request) admission.Response {
	if (req.RequestKind.Kind == "Binding") && (req.Operation == "CREATE") {
		if debug {
			fmt.Printf("Pod %s/%s raw binding:\n%s\n",
				req.Namespace, req.Name, string(req.Object.Raw))
		}

		binding := new(v1.Binding)
		err := json.Unmarshal(req.Object.Raw, binding)
		if err != nil {
			return webhook.Errored(400, fmt.Errorf("json unmarshal Binding with error: %v", err))
		}

		if binding.Target.Kind != "Node" || binding.Target.Name == "" {
			fmt.Printf("Pod %s/%s binding target is not Node or target name empty\n", binding.Namespace, binding.Name)
			return webhook.Allowed("skipped")
		}

		fmt.Printf("Pod %s/%s binding to Node %v\n",
			binding.Namespace, binding.Name, binding.Target.Name)

		node := new(v1.Node)
		err = m.cli.Get(ctx, client.ObjectKey{Name: binding.Target.Name}, node)
		if err != nil {
			return webhook.Errored(400, fmt.Errorf("failed to get node: %v", err))
		}

		pod := new(v1.Pod)
		err = m.cli.Get(ctx, client.ObjectKey{Namespace: binding.Namespace, Name: binding.Name}, pod)
		if err != nil {
			return webhook.Errored(400, fmt.Errorf("failed to get pod"))
		}

		oldPod := pod.DeepCopy()

		nodeLabels := node.GetLabels()
		topologyLabels := make(map[string]string)
		for key, value := range nodeLabels {
			if strings.HasPrefix(key, "topology.kubernetes.io/") {
				topologyLabels[key] = value
			}
		}
		podLabels := pod.GetLabels()
		if podLabels == nil {
			podLabels = make(map[string]string)
		}
		labelsEqual := true
		for key, value := range topologyLabels {
			if podValue, exists := podLabels[key]; !exists || podValue != value {
				labelsEqual = false
				break
			}
		}
		if labelsEqual {
			fmt.Printf("Pod %s/%s labels already updated\n", pod.Namespace, pod.Name)
			return webhook.Allowed("skipped")
		}
		for key, value := range topologyLabels {
			if debug {
				fmt.Printf("Pod %s/%s label %s updated to %s\n", pod.Namespace, pod.Name, key, value)
			}
			podLabels[key] = value
		}
		pod.SetLabels(podLabels)
		p := client.StrategicMergeFrom(oldPod)
		err = m.cli.Patch(ctx, pod, p)
		if err != nil {
			return webhook.Errored(500, fmt.Errorf("failed to patch pod: %v", err))
		}
		fmt.Printf("Pod %s/%s labels updated\n", pod.Namespace, pod.Name)
	}

	return webhook.Allowed("skipped")
}
