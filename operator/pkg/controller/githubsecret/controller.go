package githubsecret

import (
	"context"

	"github.com/google/go-github/v45/github"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	githubv1 "github.com/Qalisa/push-github-secrets-operator/pkg/apis/github/v1"
)

// Reconciler reconciles GithubOrganizationWatch objects
type Reconciler struct {
	client.Client
	githubClient *github.Client
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the GithubOrganizationWatch instance
	config := &githubv1.GithubOrganizationWatch{}
	if err := r.Get(ctx, req.NamespacedName, config); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Get ConfigMap
	configMap := &corev1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: config.Namespace,
		Name:      config.Spec.ConfigSetRef,
	}, configMap); err != nil {
		logger.Error(err, "failed to get ConfigMap")
		return ctrl.Result{}, err
	}

	// Get repositories from GitHub
	repos, _, err := r.githubClient.Repositories.ListByOrg(ctx, config.Spec.Organization, &github.RepositoryListByOrgOptions{})
	if err != nil {
		logger.Error(err, "failed to list repositories")
		return ctrl.Result{}, err
	}

	// Process matching repositories
	for _, repo := range repos {
		if hasMatchingTopics(repo.Topics, config.Spec.Topics) {
			if err := r.syncSecretsToRepo(ctx, repo, configMap.Data); err != nil {
				logger.Error(err, "failed to sync secrets", "repository", repo.GetName())
				continue
			}
		}
	}

	return ctrl.Result{}, nil
}

func hasMatchingTopics(repoTopics []string, requiredTopics []string) bool {
	topicMap := make(map[string]bool)
	for _, topic := range repoTopics {
		topicMap[topic] = true
	}

	for _, required := range requiredTopics {
		if !topicMap[required] {
			return false
		}
	}
	return true
}

func (r *Reconciler) syncSecretsToRepo(ctx context.Context, repo *github.Repository, data map[string]string) error {
	// Implement GitHub Actions secret syncing logic here
	return nil
}
