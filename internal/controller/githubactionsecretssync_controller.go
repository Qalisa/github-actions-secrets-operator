// secrets_sync_controller.go

package controller

import (
	"context"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	qalisav1alpha1 "github.com/qalisa/push-github-secrets-operator/api/v1alpha1"
	"github.com/qalisa/push-github-secrets-operator/internal/utils"
	"github.com/qalisa/push-github-secrets-operator/pkg/github"
)

type GithubActionSecretsSyncReconciler struct {
	client.Client
	*runtime.Scheme
	*sync.RWMutex
	GitHubClient github.Client
}

// +kubebuilder:rbac:groups=qalisa.qalisa.github.io,resources=githubactionsecretssyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=qalisa.qalisa.github.io,resources=githubactionsecretssyncs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=qalisa.qalisa.github.io,resources=githubactionsecretssyncs/finalizers,verbs=update
// +kubebuilder:rbac:groups=qalisa.qalisa.github.io,resources=githubsyncrepoes,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

func (r *GithubActionSecretsSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if !r.RWMutex.TryLock() {
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}
	defer r.RWMutex.Unlock()

	//
	// Try to get instance of CRD
	//

	instance := &qalisav1alpha1.GithubActionSecretsSync{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		// Do not exist anymore ?
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		// any other kind of error
		return ctrl.Result{}, err
	}

	//
	// Fill sync buffer
	//

	dataBySync := utils.SecVarsBySync{}
	success := utils.FillSyncBuffer(ctx, r.Client, instance, &dataBySync)
	if !success {
		return ctrl.Result{}, nil
	}

	//
	// Filter from all repo configs which that are concerned
	//

	// Initialize the list of resources
	var allRepoConfigs qalisav1alpha1.GithubSyncRepoList
	var toApplyTo []qalisav1alpha1.GithubSyncRepo

	// List all resources of the specified type
	if err := r.List(ctx, &allRepoConfigs, &client.ListOptions{}); err != nil {
		// Handle the error
		return ctrl.Result{}, err
	}

	for _, repo := range allRepoConfigs.Items {
		if utils.Contains(repo.Spec.SecretsSyncRefs, instance.Name) {
			toApplyTo = append(toApplyTo, repo)
		}
	}

	//
	//
	//

	return utils.SynchronizeToGithub(ctx, r.Client, r.GitHubClient, toApplyTo, dataBySync)
}

func (r *GithubActionSecretsSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&qalisav1alpha1.GithubActionSecretsSync{}).
		Named("githubactionsecretssync").
		Complete(r)
}
