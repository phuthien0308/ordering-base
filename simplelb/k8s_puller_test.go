package simplelb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestK8sAddressPuller_Pull(t *testing.T) {
	ctx := context.TODO()
	namespace := "default"
	serviceName := "my-service"

	t.Run("success with ports", func(t *testing.T) {
		client := fake.NewSimpleClientset(&v1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: namespace,
			},
			Subsets: []v1.EndpointSubset{
				{
					Addresses: []v1.EndpointAddress{
						{IP: "10.0.0.1"},
						{IP: "10.0.0.2"},
					},
					Ports: []v1.EndpointPort{
						{Port: 8080},
					},
				},
			},
		})

		puller := NewK8sAddressPuller(client, namespace)
		addrs, err := puller.Pull(ctx, serviceName)
		assert.NoError(t, err)
		assert.ElementsMatch(t, []Address{"10.0.0.1:8080", "10.0.0.2:8080"}, addrs)
	})

	t.Run("success without ports", func(t *testing.T) {
		client := fake.NewSimpleClientset(&v1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: namespace,
			},
			Subsets: []v1.EndpointSubset{
				{
					Addresses: []v1.EndpointAddress{
						{IP: "10.0.0.1"},
					},
				},
			},
		})

		puller := NewK8sAddressPuller(client, namespace)
		addrs, err := puller.Pull(ctx, serviceName)
		assert.NoError(t, err)
		assert.ElementsMatch(t, []Address{"10.0.0.1"}, addrs)
	})

	t.Run("no endpoints", func(t *testing.T) {
		client := fake.NewSimpleClientset() // Empty client
		puller := NewK8sAddressPuller(client, namespace)
		addrs, err := puller.Pull(ctx, serviceName)
		assert.Error(t, err) // Should error because Endpoints resource doesn't exist
		assert.Nil(t, addrs)
	})

	t.Run("empty subsets", func(t *testing.T) {
		client := fake.NewSimpleClientset(&v1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: namespace,
			},
			Subsets: []v1.EndpointSubset{},
		})

		puller := NewK8sAddressPuller(client, namespace)
		addrs, err := puller.Pull(ctx, serviceName)
		assert.NoError(t, err)
		assert.Empty(t, addrs)
	})
}
