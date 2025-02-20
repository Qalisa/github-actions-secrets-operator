package utils

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	qalisav1alpha1 "github.com/qalisa/github-actions-secrets-operator/api/v1alpha1"
	"github.com/qalisa/github-actions-secrets-operator/pkg/github"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: handle timeouts, requeue with "return ctrl.Result{RequeueAfter: time.Minute}, nil" ?
func SynchronizeToGithub(ctx context.Context, cli client.Client, logger logr.Logger, ghCli github.Client, toApplyTo []*qalisav1alpha1.GithubSyncRepo, secVarsToSync SecVarsBySync) (ctrl.Result, error) {

	//
	//
	//

	if len(toApplyTo) > 0 {
		logger.Info("Start checking sync status")

	} else {
		logger.Info("No repositories to sync, nothing to do.")
		return ctrl.Result{}, nil
	}

	//
	//
	//

	secVarTypes := []GithubActionSecVarType{Variable, Secret}
	var ghPropsSyncStateDict *[]qalisav1alpha1.GithubPropertySyncState

	//
	//
	// for each repository to sync...
	for _, repoCRD := range toApplyTo {
		//
		var resultStatsStr string

		//
		syncAttempts := SyncAttemptsByType{}
		for _, sType := range secVarTypes {
			syncAttemptsOfType := SyncAttempts{}
			syncAttempts[sType] = &syncAttemptsOfType
		}

		//
		// Try to parse repo
		//
		repo, err := ParseRepository(*repoCRD)
		if err != nil {
			SetSyncedStatusCondition(repoCRD, &repoCRD.Status.Conditions, "False", err.Error())
			// if failed, skip syncing altogether
			goto doRegisterStatus
		}

		logger.Info("Checking...", "repo", repo)

		//
		for syncTypeInt := range secVarTypes {
			//
			//
			//

			//
			syncType := GithubActionSecVarType(syncTypeInt)

			//
			ghPropsSyncStateDict = syncType.AssociatedSyncState(repoCRD)

			//
			syncAttemptsOfType := syncAttempts[syncType]

			//
			for _, propertiesBucket := range secVarsToSync[syncType] {
				for propertytName, secVar := range propertiesBucket {
					//
					syncAttemptsOfType.BumpTotal()

					// try to find if already synced
					if isGHPropertyAlreadySynced(ghPropsSyncStateDict, propertytName, secVar) {
						syncAttemptsOfType.BumpNotNeeded()
						logger.Info("Already synced against API with the same value, skipping.",
							"repo", repo,
							syncType.String(), propertytName,
						)
						continue
					}

					//
					logger.Info("Attempting sync...",
						"repo", repo,
						syncType.String(), propertytName,
					)

					// if not, try to update w/ Github API
					err := secVar.UpdateAgainstGithubApiAs(ctx, ghCli, syncType, repo, propertytName)

					if err != nil {
						logger.Info("Failed to sync against Github API",
							"repo", repo,
							syncType.String(), propertytName,
							"error", err,
						)
					} else {
						logger.Info("Successful synced against Github API",
							"repo", repo,
							syncType.String(), propertytName,
						)
					}

					// whatever the result, define sync state
					defineGHPropertySyncStatus(repoCRD, ghPropsSyncStateDict, propertytName, secVar, err, syncAttemptsOfType)
				}
			}
		}

		//
		//
		//

		//
		resultStatsStr = SyncAttempts_ProduceStats(syncAttempts)
		logger.Info("Repo sync attempt finished",
			"repo", repo,
			"recap", resultStatsStr,
		)

		//
		if SyncAttempts_AnyHasFailed(syncAttempts) {
			SetSyncedStatusCondition(repoCRD, &repoCRD.Status.Conditions, "False", fmt.Sprintf("Some synchronizations failed %s", resultStatsStr))
		} else {
			SetSyncedStatusCondition(repoCRD, &repoCRD.Status.Conditions, "True", fmt.Sprintf("All properties synced %s", resultStatsStr))
		}

		//
		//
		//

	doRegisterStatus:
		// now, try to update status
		if err := cli.Status().Update(ctx, repoCRD); err != nil {
			logger.Error(err, "Unexpected fatal error while saving status for current GithubSyncRepo; rescheduling reconciliation.")
			return ctrl.Result{}, err
		}
	}

	logger.Info("Sync run ended")

	//
	return ctrl.Result{}, nil
}
