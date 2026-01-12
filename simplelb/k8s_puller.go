package simplelb

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sAddressPuller struct {
	client    kubernetes.Interface
	namespace string
}

func NewK8sAddressPuller(client kubernetes.Interface, namespace string) *K8sAddressPuller {
	return &K8sAddressPuller{
		client:    client,
		namespace: namespace,
	}
}

// Pull implements AddressPuller.
func (k *K8sAddressPuller) Pull(ctx context.Context, serviceName string) ([]Address, error) {
	endpoints, err := k.client.CoreV1().Endpoints(k.namespace).Get(ctx, serviceName, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoints for service %s: %w", serviceName, err)
	}

	return extractAddresses(endpoints), nil
}

func extractAddresses(endpoints *corev1.Endpoints) []Address {
	var addresses []Address
	for _, subset := range endpoints.Subsets {
		for _, addr := range subset.Addresses {
			// In a real scenario, you might want to combine IP with specific ports.
			// For this implementation, we'll just use the IP.
			// If ports are needed, we would iterate subset.Ports as well.
			if len(subset.Ports) > 0 {
				for _, port := range subset.Ports {
					addresses = append(addresses, Address(fmt.Sprintf("%s:%d", addr.IP, port.Port)))
				}
			} else {
				addresses = append(addresses, Address(addr.IP))
			}
		}
	}
	return addresses
}
