package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SetStatusCondition updates the status condition of the resource
func SetStatusCondition(instance metav1.Object, conditions *[]metav1.Condition, status, message string) {
	condition := metav1.Condition{
		Type:               "Synced",
		Status:             metav1.ConditionStatus(status),
		ObservedGeneration: instance.GetGeneration(),
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             strings.ReplaceAll(status, " ", ""),
		Message:            message,
	}

	// Update or append the condition
	for i, existingCondition := range *conditions {
		if existingCondition.Type == condition.Type {
			(*conditions)[i] = condition
			return
		}
	}
	*conditions = append(*conditions, condition)
}

// ParseRepository splits a repository string in the format "owner/repo" into owner and repo parts
func ParseRepository(repository string) (string, string, error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository format: %s, expected owner/repo", repository)
	}
	return parts[0], parts[1], nil
}

// GetSecret fetches a secret from the Kubernetes API
func GetSecret(ctx context.Context, c client.Client, namespace, name string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// GetConfigMap fetches a configmap from the Kubernetes API
func GetConfigMap(ctx context.Context, c client.Client, namespace, name string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, configMap)
	if err != nil {
		return nil, err
	}
	return configMap, nil
}
