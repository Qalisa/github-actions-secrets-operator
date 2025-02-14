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

// +kubebuilder:rbac:groups=qalisa.github.io,resources=githubsyncrepoes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=qalisa.github.io,resources=githubsyncrepoes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=qalisa.github.io,resources=githubsyncrepoes/finalizers,verbs=update
// +kubebuilder:rbac:groups=qalisa.github.io,resources=githubactionsecretssyncs,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

func (r *GithubSyncRepoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.RWMutex.Lock()
	defer r.RWMutex.Unlock()

	// defer func() {
	// 	if r := recover(); r != nil {
	// 		fmt.Printf("Recovered from panic: %v\n", r)
	// 	}
	// }()

	//
	//
	//

	var syncErr error
	var result ctrl.Result
	instance := &qalisav1alpha1.GithubSyncRepo{}
	toApplyTo := []qalisav1alpha1.GithubSyncRepo{*instance}
	var dataBySync utils.SecVarsBySync
	var concernedSyncConfigs []qalisav1alpha1.GithubActionSecretsSync
	var tempSyncConfigs qalisav1alpha1.GithubActionSecretsSyncList
	reachedSync := false

	//
	// try to get instance of CRD
	//

	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		// Do not exist anymore ?
		if errors.IsNotFound(err) {
			//
			// TODO: HANDLE DELETION of resource
			//
			goto doRegisterStatus
		}

		// any other kind of error. Would immediately schedule requeue because of err is set
		return ctrl.Result{}, err
	}

	//
	// test parsing of repo name
	//

	if _, err := utils.ParseRepository(*instance); err != nil {
		utils.SetSyncedStatusCondition(instance, &instance.Status.Conditions, "False", err.Error())
		goto doRegisterStatus
	}

	//
	// Find concerned Syncs
	//

	// List all resources with specific names
	for _, name := range instance.Spec.SecretsSyncRefs {
		// find with refd name
		if err := r.List(ctx, &tempSyncConfigs, client.MatchingFields{"metadata.name": name}); err != nil {
			utils.SetSyncedStatusCondition(instance, &instance.Status.Conditions, "False", err.Error())
			goto doRegisterStatus
		}

		// if not finding exactly 1 ref
		found := len(tempSyncConfigs.Items)
		if found != 1 {
			errMsg := fmt.Sprintf("Failed to find referenced GithubActionSecretsSync '%s' within cluster (found %d)", name, found)
			utils.SetSyncedStatusCondition(instance, &instance.Status.Conditions, "False", errMsg)
			goto doRegisterStatus
		}

		// append first to concerned
		concernedSyncConfigs = append(concernedSyncConfigs, tempSyncConfigs.Items[0])
	}

	//
	// Fill sync buffer
	//

	for _, sync := range concernedSyncConfigs {
		if err := utils.FillSyncBuffer(ctx, r.Client, &sync, &dataBySync); err != nil {
			utils.SetSyncedStatusCondition(instance, &instance.Status.Conditions, "False", err.Error())
			goto doRegisterStatus
		}
	}

	//
	//
	//

	result, syncErr = utils.SynchronizeToGithub(ctx, r.Client, r.GitHubClient, toApplyTo, dataBySync)
	reachedSync = true

	//
	//
	//

doRegisterStatus:
	if !reachedSync {
		// now, try to update this instance's status
		if err := r.Client.Status().Update(ctx, instance); err != nil {
			// Kind of anormal error; Would immediately schedule requeue because of err is set
			return ctrl.Result{}, err
		}
	}

	//
	return result, syncErr
}

func (r *GithubSyncRepoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// ✅ Add the field index for "metadata.name"
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &qalisav1alpha1.GithubSyncRepo{}, "metadata.name", func(rawObj client.Object) []string {
		obj := rawObj.(*qalisav1alpha1.GithubSyncRepo) // Type assertion
		return []string{obj.Name}                      // Index by Name
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&qalisav1alpha1.GithubSyncRepo{}).
		Named("githubsyncrepo").
		Complete(r)
}
