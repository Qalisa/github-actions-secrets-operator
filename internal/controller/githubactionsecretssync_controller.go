/*
Copyright 2025 Guillaume Vara.

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

package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	qalisav1alpha1 "github.com/qalisa/push-github-secrets-operator/api/v1alpha1"
	"github.com/qalisa/push-github-secrets-operator/pkg/github"
)

// GithubActionSecretsSyncReconciler reconciles a GithubActionSecretsSync object
type GithubActionSecretsSyncReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	GitHubClient github.Client
}

// +kubebuilder:rbac:groups=qalisa.qalisa.github.io,resources=githubactionsecretssyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=qalisa.qalisa.github.io,resources=githubactionsecretssyncs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=qalisa.qalisa.github.io,resources=githubactionsecretssyncs/finalizers,verbs=update
// +kubebuilder:rbac:groups=qalisa.qalisa.github.io,resources=githubsyncrepoes,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile handles the synchronization of GitHub Actions secrets and variables
func (r *GithubActionSecretsSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Starting reconciliation", "namespacedName", req.NamespacedName)

	// Fetch the GithubActionSecretsSync instance
	instance := &qalisav1alpha1.GithubActionSecretsSync{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Initialize status conditions if they don't exist
	if instance.Status.Conditions == nil {
		instance.Status.Conditions = []metav1.Condition{}
	}

	// Get GithubSyncRepo instances that reference this sync config
	repoList := &qalisav1alpha1.GithubSyncRepoList{}
	if err := r.List(ctx, repoList); err != nil {
		r.setStatusCondition(instance, "Failed", fmt.Sprintf("Failed to list repos: %v", err))
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Process secrets
	for _, secretRef := range instance.Spec.Secrets {
		secret := &corev1.Secret{}
		err := r.Get(ctx, types.NamespacedName{Name: secretRef.SecretRef, Namespace: req.Namespace}, secret)
		if err != nil {
			r.setStatusCondition(instance, "Failed", fmt.Sprintf("Failed to get secret %s: %v", secretRef.SecretRef, err))
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}

		secretValue, exists := secret.Data[secretRef.Key]
		if !exists {
			r.setStatusCondition(instance, "Failed", fmt.Sprintf("Key %s not found in secret %s", secretRef.Key, secretRef.SecretRef))
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}

		githubSecretName := secretRef.GithubSecretName
		if githubSecretName == "" {
			githubSecretName = secretRef.Key
		}

		// Sync secret to all referenced repositories
		for _, repo := range repoList.Items {
			for _, syncRef := range repo.Spec.SecretsSyncRefs {
				if syncRef == instance.Name {
					owner, repoName, err := parseRepository(repo.Spec.Repository)
					if err != nil {
						logger.Error(err, "Failed to parse repository", "repository", repo.Spec.Repository)
						continue
					}

					if err := r.GitHubClient.CreateOrUpdateSecret(ctx, owner, repoName, githubSecretName, secretValue); err != nil {
						logger.Error(err, "Failed to sync secret", "repository", repo.Spec.Repository)
						continue
					}

					logger.Info("Successfully synced secret", "name", githubSecretName, "repository", repo.Spec.Repository)
				}
			}
		}
	}

	// Process variables
	for _, varRef := range instance.Spec.Variables {
		configMap := &corev1.ConfigMap{}
		err := r.Get(ctx, types.NamespacedName{Name: varRef.ConfigMapRef, Namespace: req.Namespace}, configMap)
		if err != nil {
			r.setStatusCondition(instance, "Failed", fmt.Sprintf("Failed to get configmap %s: %v", varRef.ConfigMapRef, err))
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}

		varValue, exists := configMap.Data[varRef.Key]
		if !exists {
			r.setStatusCondition(instance, "Failed", fmt.Sprintf("Key %s not found in configmap %s", varRef.Key, varRef.ConfigMapRef))
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}

		githubVarName := varRef.GithubVariableName
		if githubVarName == "" {
			githubVarName = varRef.Key
		}

		// Sync variable to all referenced repositories
		for _, repo := range repoList.Items {
			for _, syncRef := range repo.Spec.SecretsSyncRefs {
				if syncRef == instance.Name {
					owner, repoName, err := parseRepository(repo.Spec.Repository)
					if err != nil {
						logger.Error(err, "Failed to parse repository", "repository", repo.Spec.Repository)
						continue
					}

					if err := r.GitHubClient.CreateOrUpdateVariable(ctx, owner, repoName, githubVarName, varValue); err != nil {
						logger.Error(err, "Failed to sync variable", "repository", repo.Spec.Repository)
						continue
					}

					logger.Info("Successfully synced variable", "name", githubVarName, "repository", repo.Spec.Repository)
				}
			}
		}
	}

	// Update status
	r.setStatusCondition(instance, "Synced", "Successfully processed all secrets and variables")
	instance.Status.LastSyncTime = &metav1.Time{Time: time.Now()}
	if err := r.Status().Update(ctx, instance); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	return ctrl.Result{RequeueAfter: time.Hour}, nil
}

// setStatusCondition updates the status condition of the GithubActionSecretsSync instance
func (r *GithubActionSecretsSyncReconciler) setStatusCondition(instance *qalisav1alpha1.GithubActionSecretsSync, status, message string) {
	condition := metav1.Condition{
		Type:               "Synced",
		Status:             metav1.ConditionStatus(status),
		ObservedGeneration: instance.Generation,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             strings.ReplaceAll(status, " ", ""),
		Message:            message,
	}

	// Update or append the condition
	for i, existingCondition := range instance.Status.Conditions {
		if existingCondition.Type == condition.Type {
			instance.Status.Conditions[i] = condition
			return
		}
	}
	instance.Status.Conditions = append(instance.Status.Conditions, condition)
}

// parseRepository splits a repository string in the format "owner/repo" into owner and repo parts
func parseRepository(repository string) (string, string, error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository format: %s, expected owner/repo", repository)
	}
	return parts[0], parts[1], nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GithubActionSecretsSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&qalisav1alpha1.GithubActionSecretsSync{}).
		Named("githubactionsecretssync").
		Complete(r)
}
