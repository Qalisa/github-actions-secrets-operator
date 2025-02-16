package utils

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"

	qalisav1alpha1 "github.com/qalisa/github-actions-secrets-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//
//
//

func HashBytes(s []byte) uint32 {
	h := fnv.New32a()
	h.Write(s)
	return h.Sum32()
}

// Helper function to check if a value exists in an array
func Contains(arr []string, target string) bool {
	for _, item := range arr {
		if item == target {
			return true
		}
	}
	return false
}

//
//
//

type GithubRepository struct {
	Org  string
	Name string
}

// ParseRepository splits a repository string in the format "owner/repo" into owner and repo parts
func ParseRepository(repository qalisav1alpha1.GithubSyncRepo) (GithubRepository, error) {
	//
	toParse := repository.Spec.Repository
	parts := strings.Split(toParse, "/")

	//
	if len(parts) != 2 {
		return GithubRepository{}, fmt.Errorf("invalid repository format detected ('%s'), expected owner/repo", toParse)
	}

	//
	return GithubRepository{parts[0], parts[1]}, nil
}

//
//
//

// GetSecret fetches a secret from the Kubernetes API
func GetSecret(ctx context.Context, c client.Client, ref qalisav1alpha1.ResourceRef) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := c.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// GetConfigMap fetches a configmap from the Kubernetes API
func GetConfigMap(ctx context.Context, c client.Client, ref qalisav1alpha1.ResourceRef) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	err := c.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, configMap)
	if err != nil {
		return nil, err
	}
	return configMap, nil
}
