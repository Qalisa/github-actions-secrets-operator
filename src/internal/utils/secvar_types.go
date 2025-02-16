package utils

import (
	"context"
	"fmt"

	qalisav1alpha1 "github.com/qalisa/github-actions-secrets-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GithubActionSecVarType int

const (
	Variable GithubActionSecVarType = iota
	Secret   GithubActionSecVarType = iota
)

func (cs GithubActionSecVarType) String() string {
	switch cs {
	case Variable:
		return "Variable"
	case Secret:
		return "Secret"
	}
	panic("Unexpected GithubActionSecVarType type")
}

func (cs GithubActionSecVarType) AssociatedSyncState(repo *qalisav1alpha1.GithubSyncRepo) *[]qalisav1alpha1.GithubPropertySyncState {
	switch cs {
	case Variable:
		return &repo.Status.VariablesSyncStates
	case Secret:
		return &repo.Status.SecretsSyncStates
	}
	panic("Unexpected GithubActionSecVarType type")
}

//
//
//

// {Variable|Secret}:<GithubActionSecretsSync:name>:<{Variable|Secret}:gh-name>:(value&hash(value))
type SecVarsBySync map[GithubActionSecVarType]map[types.NamespacedName]map[string]SecVar

// SetSecVar safely initializes the nested maps and sets the SecVar value.
func SafeSetSecVar(svs *SecVarsBySync, secVarType GithubActionSecVarType, source metav1.ObjectMeta, ghPropertyName string, secVar SecVar) {
	if *svs == nil {
		*svs = make(SecVarsBySync) // Ensure the outer map is initialized
	}

	// Ensure the first-level map is initialized
	if (*svs)[secVarType] == nil {
		(*svs)[secVarType] = make(map[types.NamespacedName]map[string]SecVar)
	}

	// Ensure the second-level map is initialized
	nsName := types.NamespacedName{Namespace: source.Namespace, Name: source.Name}
	if (*svs)[secVarType][nsName] == nil {
		(*svs)[secVarType][nsName] = make(map[string]SecVar)
	}

	// Assign the value
	(*svs)[secVarType][nsName][ghPropertyName] = secVar
}

//
//
//

func FillSyncBuffer(ctx context.Context, c client.Client, instance *qalisav1alpha1.GithubActionSecretsSync, dataBySync *SecVarsBySync) error {
	// Process secrets
	for _, secretRef := range instance.Spec.Secrets {
		// Get Secret
		secret, err := GetSecret(ctx, c, secretRef.SecretRef)
		if err != nil {
			return fmt.Errorf("failed to get secret '%s' in namespace '%s': %v", secretRef.SecretRef, instance.Namespace, err)
		}

		// checks for key
		secretValue, exists := secret.Data[secretRef.Key]
		if !exists {
			return fmt.Errorf("key %s not found in secret %s", secretRef.Key, secretRef.SecretRef)
		}

		//
		githubSecretName := secretRef.GithubSecretName
		if githubSecretName == "" {
			githubSecretName = secretRef.Key
		}

		//
		SafeSetSecVar(dataBySync, Secret, instance.ObjectMeta, githubSecretName, SecVar{
			Value:       secretValue,
			HashOfValue: HashBytes(secretValue),
		})
	}

	// Process variables
	for _, configMapRef := range instance.Spec.Variables {
		// Get Secret
		configMap, err := GetConfigMap(ctx, c, configMapRef.ConfigMapRef)
		if err != nil {
			return fmt.Errorf("failed to get Config Map '%s' in namespace '%s': %v", configMapRef.ConfigMapRef, instance.Namespace, err)
		}

		// checks for key
		configValue, exists := configMap.Data[configMapRef.Key]
		if !exists {
			return fmt.Errorf("key %s not found in config map %s", configMapRef.Key, configMapRef.ConfigMapRef)

		}

		//
		githubVariableName := configMapRef.GithubVariableName
		if githubVariableName == "" {
			githubVariableName = configMapRef.Key
		}

		//
		configValueAsBytes := []byte(configValue)
		SafeSetSecVar(dataBySync, Variable, instance.ObjectMeta, githubVariableName, SecVar{
			Value:       configValueAsBytes,
			HashOfValue: HashBytes(configValueAsBytes),
		})
	}

	//
	return nil
}
