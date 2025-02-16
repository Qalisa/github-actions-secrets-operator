package utils

import (
	"strings"
	"time"

	qalisav1alpha1 "github.com/qalisa/github-actions-secrets-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SetSyncedStatusCondition(instance metav1.Object, conditions *[]metav1.Condition, status, message string) {
	setStatusCondition(instance, conditions, "Synced", status, message)
}

// Updates the status condition of the resource
func setStatusCondition(instance metav1.Object, conditions *[]metav1.Condition, statusType, status, message string) {
	condition := metav1.Condition{
		Type:               statusType,
		Status:             metav1.ConditionStatus(status),
		ObservedGeneration: instance.GetGeneration(),
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             strings.ReplaceAll(status, " ", ""),
		Message:            message,
	}

	// Update or append the condition
	for i, existingCondition := range *conditions {
		if existingCondition.Type == condition.Type {
			(*conditions)[i] = condition
			return
		}
	}
	*conditions = append(*conditions, condition)
}

func getStatusCondition(conditions *[]metav1.Condition, statusType string) *metav1.Condition {
	for i, existingCondition := range *conditions {
		if existingCondition.Type == statusType {
			return &(*conditions)[i]
		}
	}

	return nil
}

func getSyncedStatusCondition(conditions *[]metav1.Condition) *metav1.Condition {
	return getStatusCondition(conditions, "Synced")
}

//
//
//

func findGHPropertyStateConditions(states *[]qalisav1alpha1.GithubPropertySyncState, githubPropertyName string) *[]metav1.Condition {
	for i, state := range *states {
		if state.GithubPropertyName == githubPropertyName {
			return &(*states)[i].Conditions
		}
	}
	return nil
}

func isGHPropertyAlreadySynced(states *[]qalisav1alpha1.GithubPropertySyncState, githubPropertyName string, secvar SecVar) bool {
	conditions := findGHPropertyStateConditions(states, githubPropertyName)

	// if no conditions, means nothing has ever synced
	if conditions == nil {
		return false
	}

	//
	return secvar.isSyncedFrom(conditions)
}

//
//
//

func defineGHPropertySyncStatus(instance metav1.Object, states *[]qalisav1alpha1.GithubPropertySyncState, githubPropertyName string, secvar SecVar, err error, syncAttempts *SyncAttempts) {
	conditions := findGHPropertyStateConditions(states, githubPropertyName)

	// means we need to create
	if conditions == nil {
		*states = append(*states, qalisav1alpha1.GithubPropertySyncState{
			GithubPropertyName: githubPropertyName,
			Conditions:         []metav1.Condition{},
		})

		//
		conditions = findGHPropertyStateConditions(states, githubPropertyName)
	}

	secvar.defineSyncStatusFrom(instance, conditions, err, syncAttempts)
}
