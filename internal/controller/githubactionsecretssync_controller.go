// secrets_sync_controller.go

package controller

import (
	"context"
	"sync"

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

// +kubebuilder:rbac:groups=qalisa.github.io,resources=githubactionsecretssyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=qalisa.github.io,resources=githubactionsecretssyncs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=qalisa.github.io,resources=githubactionsecretssyncs/finalizers,verbs=update
// +kubebuilder:rbac:groups=qalisa.github.io,resources=githubsyncrepoes,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

func (r *GithubActionSecretsSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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
	var dataBySync utils.SecVarsBySync
	var allRepoConfigs qalisav1alpha1.GithubSyncRepoList
	var toApplyTo []qalisav1alpha1.GithubSyncRepo

	//
	// Try to get instance of CRD
	//

	instance := &qalisav1alpha1.GithubActionSecretsSync{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		// Do not exist anymore ?
		if errors.IsNotFound(err) {
			//
			// TODO: HANDLE DELETION of resource
			//
			goto doRegisterStatus
		}

		// any other kind of error, which is alarming. Would immediately schedule requeue because of err is set
		return ctrl.Result{}, err
	}

	//
	// Fill sync buffer
	//

	if err := utils.FillSyncBuffer(ctx, r.Client, instance, &dataBySync); err != nil {
		utils.SetSyncedStatusCondition(instance, &instance.Status.Conditions, "False", err.Error())
		goto doRegisterStatus
	}

	//
	// Filter from all repo configs which that are concerned
	//

	// List all resources of the specified type
	if err := r.List(ctx, &allRepoConfigs, &client.ListOptions{}); err != nil {
		utils.SetSyncedStatusCondition(instance, &instance.Status.Conditions, "False", err.Error())
		goto doRegisterStatus
	}

	for _, repo := range allRepoConfigs.Items {
		if utils.Contains(repo.Spec.SecretsSyncRefs, instance.Name) {
			toApplyTo = append(toApplyTo, repo)
		}
	}

	//
	//
	//

	result, syncErr = utils.SynchronizeToGithub(ctx, r.Client, r.GitHubClient, toApplyTo, dataBySync)

	//
	//
	//

doRegisterStatus:
	// now, try to update this instance's status
	if err := r.Client.Status().Update(ctx, instance); err != nil {
		// Kind of anormal error; Would immediately schedule requeue because of err is set
		return ctrl.Result{}, err
	}

	//
	return result, syncErr
}

func (r *GithubActionSecretsSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&qalisav1alpha1.GithubActionSecretsSync{}).
		Named("githubactionsecretssync").
		Complete(r)
}
