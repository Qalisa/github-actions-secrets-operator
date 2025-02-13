// repo_sync_controller.go

package controller

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	qalisav1alpha1 "github.com/qalisa/push-github-secrets-operator/api/v1alpha1"
	"github.com/qalisa/push-github-secrets-operator/internal/utils"
	"github.com/qalisa/push-github-secrets-operator/pkg/github"
)

type GithubSyncRepoReconciler struct {
	client.Client
	*runtime.Scheme
	*sync.RWMutex
	GitHubClient github.Client
}

// +kubebuilder:rbac:groups=qalisa.qalisa.github.io,resources=githubsyncrepoes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=qalisa.qalisa.github.io,resources=githubsyncrepoes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=qalisa.qalisa.github.io,resources=githubsyncrepoes/finalizers,verbs=update
// +kubebuilder:rbac:groups=qalisa.qalisa.github.io,resources=githubactionsecretssyncs,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

func (r *GithubSyncRepoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.RWMutex.Lock()
	defer r.RWMutex.Unlock()

	//
	// try to get instance of CRD
	//

	instance := &qalisav1alpha1.GithubSyncRepo{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		// Do not exist anymore ?
		if errors.IsNotFound(err) {
			//
			// TODO: HANDLE DELETION of resource
			//
			return ctrl.Result{}, nil
		}

		// any other kind of error. Would immediately schedule requeue because of err is set
		return ctrl.Result{}, err
	}

	//
	// test parsing of repo name
	//

	_, err := utils.ParseRepository(*instance)
	if err != nil {
		utils.SetSyncedStatusCondition(instance, &instance.Status.Conditions, "False", err.Error())
		return ctrl.Result{}, nil
	}

	//
	// Find concerned Syncs
	//

	// Initialize the list of resources
	var concernedSyncConfigs []qalisav1alpha1.GithubActionSecretsSync

	// List all resources with specific names
	var tempSyncConfigs qalisav1alpha1.GithubActionSecretsSyncList
	for _, name := range instance.Spec.SecretsSyncRefs {
		// find with refd name
		if err := r.List(ctx, &tempSyncConfigs, client.MatchingFields{"metadata.name": name}); err != nil {
			utils.SetSyncedStatusCondition(instance, &instance.Status.Conditions, "False", err.Error())
			return ctrl.Result{}, nil
		}

		// if not finding exactly 1 ref
		found := len(tempSyncConfigs.Items)
		if found != 1 {
			errMsg := fmt.Sprintf("Failed to find referenced GithubActionSecretsSync '%s' within cluster (found %d)", name, found)
			utils.SetSyncedStatusCondition(instance, &instance.Status.Conditions, "False", errMsg)
			return ctrl.Result{}, nil
		}

		// append first to concerned
		concernedSyncConfigs = append(concernedSyncConfigs, tempSyncConfigs.Items[0])
	}

	//
	// Fill sync buffer
	//

	dataBySync := utils.SecVarsBySync{}
	for _, sync := range concernedSyncConfigs {
		if err := utils.FillSyncBuffer(ctx, r.Client, &sync, &dataBySync); err != nil {
			utils.SetSyncedStatusCondition(instance, &instance.Status.Conditions, "False", err.Error())
			return ctrl.Result{}, nil
		}
	}

	//
	//
	//

	toApplyTo := []qalisav1alpha1.GithubSyncRepo{*instance}
	return utils.SynchronizeToGithub(ctx, r.Client, r.GitHubClient, toApplyTo, dataBySync)
}

func (r *GithubSyncRepoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&qalisav1alpha1.GithubSyncRepo{}).
		Named("githubsyncrepo").
		Complete(r)
}
