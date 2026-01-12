package simplelb

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sAddressWatcher struct {
	client    kubernetes.Interface
	namespace string
}

func NewK8sAddressWatcher(client kubernetes.Interface, namespace string) *K8sAddressWatcher {
	return &K8sAddressWatcher{
		client:    client,
		namespace: namespace,
	}
}

// Watch implements AddressWatcher.
func (k *K8sAddressWatcher) Watch(ctx context.Context, serviceName string) (<-chan []Address, error) {
	watcher, err := k.client.CoreV1().Endpoints(k.namespace).Watch(ctx, v1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", serviceName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start watching endpoints for service %s: %w", serviceName, err)
	}

	resultCh := make(chan []Address)

	go func() {
		defer close(resultCh)
		defer watcher.Stop()

		for {
			select {
			case event, ok := <-watcher.ResultChan():
				if !ok {
					return
				}

				switch event.Type {
				case "ADDED", "MODIFIED", "DELETED":
					if endpoints, ok := event.Object.(*corev1.Endpoints); ok {
						resultCh <- extractAddresses(endpoints)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return resultCh, nil
}
