package utils

import (
	"context"
	"fmt"

	"github.com/qalisa/github-actions-secrets-operator/pkg/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SecVar struct {
	Value       []byte
	HashOfValue uint32
}

func (r *SecVar) hashAsString() string {
	return fmt.Sprint(r.HashOfValue)
}

func (r *SecVar) isSyncedFrom(conditions *[]metav1.Condition) bool {
	condition := getSyncedStatusCondition(conditions)

	// condition was never set, consider not synced
	if condition == nil {
		return false
	}

	// message must contain property associated value's hash
	return condition.Message == r.hashAsString()
}

// Will define the Synced status from conditions, depending on if an error is passed as argument
func (r *SecVar) defineSyncStatusFrom(instance metav1.Object, conditions *[]metav1.Condition, err error, syncAttempts *SyncAttempts) {
	if err == nil {
		SetSyncedStatusCondition(instance, conditions, "True", r.hashAsString())
		syncAttempts.BumpSuccessful()
	} else {
		SetSyncedStatusCondition(instance, conditions, "False", err.Error())
		syncAttempts.BumpFailed()
	}
}

func (r *SecVar) UpdateAgainstGithubApiAs(ctx context.Context, cli github.Client, asType GithubActionSecVarType, repo GithubRepository, ghPropName string) error {
	switch asType {
	case Variable:
		return cli.CreateOrUpdateVariable(ctx, repo.Org, repo.Name, ghPropName, string(r.Value))
	case Secret:
		return cli.CreateOrUpdateSecret(ctx, repo.Org, repo.Name, ghPropName, r.Value)
	default:
		return fmt.Errorf("undefined behavior with GithubActionSecVarType type '%d'", asType)
	}
}
