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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	qalisav1alpha1 "github.com/qalisa/push-github-secrets-operator/api/v1alpha1"
	"github.com/qalisa/push-github-secrets-operator/internal/utils"
	"github.com/qalisa/push-github-secrets-operator/pkg/github"
)

// GithubSyncRepoReconciler reconciles a GithubSyncRepo object
type GithubSyncRepoReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	GitHubClient github.Client
}

// +kubebuilder:rbac:groups=qalisa.qalisa.github.io,resources=githubsyncrepoes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=qalisa.qalisa.github.io,resources=githubsyncrepoes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=qalisa.qalisa.github.io,resources=githubsyncrepoes/finalizers,verbs=update
// +kubebuilder:rbac:groups=qalisa.qalisa.github.io,resources=githubactionsecretssyncs,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile handles the synchronization of secrets and variables to a GitHub repository
func (r *GithubSyncRepoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Starting reconciliation", "namespacedName", req.NamespacedName)

	// Fetch the GithubSyncRepo instance
	instance := &qalisav1alpha1.GithubSyncRepo{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Parse repository owner and name
	owner, repo, err := utils.ParseRepository(instance.Spec.Repository)
	if err != nil {
		utils.SetStatusCondition(instance, &instance.Status.Conditions, "False", err.Error())
		return ctrl.Result{RequeueAfter: time.Hour}, nil
	}

	// Initialize status conditions if they don't exist
	if instance.Status.Conditions == nil {
		instance.Status.Conditions = []metav1.Condition{}
	}

	// Process each referenced GithubActionSecretsSync
	for _, syncRef := range instance.Spec.SecretsSyncRefs {
		secretsSync := &qalisav1alpha1.GithubActionSecretsSync{}
		err := r.Get(ctx, types.NamespacedName{Name: syncRef}, secretsSync)
		if err != nil {
			utils.SetStatusCondition(instance, &instance.Status.Conditions, "False", fmt.Sprintf("Failed to get GithubActionSecretsSync %s: %v", syncRef, err))
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}

		// Process secrets
		for _, secretRef := range secretsSync.Spec.Secrets {
			secret, err := utils.GetSecret(ctx, r.Client, req.Namespace, secretRef.SecretRef)
			if err != nil {
				utils.SetStatusCondition(instance, &instance.Status.Conditions, "False", fmt.Sprintf("Failed to get secret %s: %v", secretRef.SecretRef, err))
				return ctrl.Result{RequeueAfter: time.Minute}, nil
			}

			value, ok := secret.Data[secretRef.Key]
			if !ok {
				utils.SetStatusCondition(instance, &instance.Status.Conditions, "False", fmt.Sprintf("Key %s not found in secret %s", secretRef.Key, secretRef.SecretRef))
				return ctrl.Result{RequeueAfter: time.Minute}, nil
			}

			githubSecretName := secretRef.GithubSecretName
			if githubSecretName == "" {
				githubSecretName = secretRef.Key
			}

			err = r.GitHubClient.CreateOrUpdateSecret(ctx, owner, repo, githubSecretName, value)
			if err != nil {
				utils.SetStatusCondition(instance, &instance.Status.Conditions, "False", fmt.Sprintf("Failed to sync secret %s: %v", githubSecretName, err))
				return ctrl.Result{RequeueAfter: time.Minute}, nil
			}
			logger.Info("Synced secret", "name", githubSecretName)
		}

		// Process variables
		for _, varRef := range secretsSync.Spec.Variables {
			configMap, err := utils.GetConfigMap(ctx, r.Client, req.Namespace, varRef.ConfigMapRef)
			if err != nil {
				utils.SetStatusCondition(instance, &instance.Status.Conditions, "False", fmt.Sprintf("Failed to get configmap %s: %v", varRef.ConfigMapRef, err))
				return ctrl.Result{RequeueAfter: time.Minute}, nil
			}

			value, ok := configMap.Data[varRef.Key]
			if !ok {
				utils.SetStatusCondition(instance, &instance.Status.Conditions, "False", fmt.Sprintf("Key %s not found in configmap %s", varRef.Key, varRef.ConfigMapRef))
				return ctrl.Result{RequeueAfter: time.Minute}, nil
			}

			githubVarName := varRef.GithubVariableName
			if githubVarName == "" {
				githubVarName = varRef.Key
			}

			err = r.GitHubClient.CreateOrUpdateVariable(ctx, owner, repo, githubVarName, value)
			if err != nil {
				utils.SetStatusCondition(instance, &instance.Status.Conditions, "False", fmt.Sprintf("Failed to sync variable %s: %v", githubVarName, err))
				return ctrl.Result{RequeueAfter: time.Minute}, nil
			}
			logger.Info("Synced variable", "name", githubVarName)
		}
	}

	// Update status
	utils.SetStatusCondition(instance, &instance.Status.Conditions, "True", "Successfully synced all secrets and variables")
	instance.Status.LastSyncTime = &metav1.Time{Time: time.Now()}
	if err := r.Status().Update(ctx, instance); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	return ctrl.Result{RequeueAfter: time.Hour}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GithubSyncRepoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&qalisav1alpha1.GithubSyncRepo{}).
		Named("githubsyncrepo").
		Complete(r)
}
