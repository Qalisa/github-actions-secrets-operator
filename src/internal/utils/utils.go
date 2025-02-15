package utils

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	qalisav1alpha1 "github.com/qalisa/github-actions-secrets-operator/api/v1alpha1"
	"github.com/qalisa/github-actions-secrets-operator/pkg/github"
	ctrl "sigs.k8s.io/controller-runtime"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//
//
//

type GithubActionSecVarType int

const (
	Variable GithubActionSecVarType = iota
	Secret   GithubActionSecVarType = iota
)

// {Variable|Secret}:<GithubActionSecretsSync:name>:<{Variable|Secret}:gh-name>:(value&hash(value))
type SecVarsBySync map[GithubActionSecVarType]map[types.NamespacedName]map[string]SecVar

//
//
//

func HashBytes(s []byte) uint32 {
	h := fnv.New32a()
	h.Write(s)
	return h.Sum32()
}

// Helper function to check if a value exists in an array
func Contains(arr []string, target string) bool {
	for _, item := range arr {
		if item == target {
			return true
		}
	}
	return false
}

//
//
//

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

type GithubRepository struct {
	Org  string
	Name string
}

// ParseRepository splits a repository string in the format "owner/repo" into owner and repo parts
func ParseRepository(repository qalisav1alpha1.GithubSyncRepo) (GithubRepository, error) {
	//
	toParse := repository.Spec.Repository
	parts := strings.Split(toParse, "/")

	//
	if len(parts) != 2 {
		return GithubRepository{}, fmt.Errorf("invalid repository format detected ('%s'), expected owner/repo", toParse)
	}

	//
	return GithubRepository{parts[0], parts[1]}, nil
}

// GetSecret fetches a secret from the Kubernetes API
func GetSecret(ctx context.Context, c client.Client, ref qalisav1alpha1.ResourceRef) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := c.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// GetConfigMap fetches a configmap from the Kubernetes API
func GetConfigMap(ctx context.Context, c client.Client, ref qalisav1alpha1.ResourceRef) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	err := c.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, configMap)
	if err != nil {
		return nil, err
	}
	return configMap, nil
}

//
//
//

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

//
//
//

func FillSyncBuffer(ctx context.Context, c client.Client, instance *qalisav1alpha1.GithubActionSecretsSync, dataBySync *SecVarsBySync) error {
	// Process secrets
	for _, secretRef := range instance.Spec.Secrets {
		// Get Secret
		secret, err := GetSecret(ctx, c, secretRef.SecretRef)
		if err != nil {
			return fmt.Errorf("failed to get secret '%s' in namespace '%s': %v", secretRef.SecretRef, instance.Namespace, err)
		}

		// checks for key
		secretValue, exists := secret.Data[secretRef.Key]
		if !exists {
			return fmt.Errorf("key %s not found in secret %s", secretRef.Key, secretRef.SecretRef)
		}

		//
		githubSecretName := secretRef.GithubSecretName
		if githubSecretName == "" {
			githubSecretName = secretRef.Key
		}

		//
		SetSecVar(dataBySync, Secret, instance.ObjectMeta, githubSecretName, SecVar{
			Value:       secretValue,
			HashOfValue: HashBytes(secretValue),
		})
	}

	// Process variables
	for _, configMapRef := range instance.Spec.Variables {
		// Get Secret
		configMap, err := GetConfigMap(ctx, c, configMapRef.ConfigMapRef)
		if err != nil {
			return fmt.Errorf("failed to get Config Map '%s' in namespace '%s': %v", configMapRef.ConfigMapRef, instance.Namespace, err)
		}

		// checks for key
		configValue, exists := configMap.Data[configMapRef.Key]
		if !exists {
			return fmt.Errorf("key %s not found in config map %s", configMapRef.Key, configMapRef.ConfigMapRef)

		}

		//
		githubVariableName := configMapRef.GithubVariableName
		if githubVariableName == "" {
			githubVariableName = configMapRef.Key
		}

		//
		configValueAsBytes := []byte(configValue)
		SetSecVar(dataBySync, Variable, instance.ObjectMeta, githubVariableName, SecVar{
			Value:       configValueAsBytes,
			HashOfValue: HashBytes(configValueAsBytes),
		})
	}

	//
	return nil
}

// SetSecVar safely initializes the nested maps and sets the SecVar value.
func SetSecVar(svs *SecVarsBySync, secVarType GithubActionSecVarType, source metav1.ObjectMeta, ghPropertyName string, secVar SecVar) {
	if *svs == nil {
		*svs = make(SecVarsBySync) // Ensure the outer map is initialized
	}

	// Ensure the first-level map is initialized
	if (*svs)[secVarType] == nil {
		(*svs)[secVarType] = make(map[types.NamespacedName]map[string]SecVar)
	}

	// Ensure the second-level map is initialized
	nsName := types.NamespacedName{Namespace: source.Namespace, Name: source.Name}
	if (*svs)[secVarType][nsName] == nil {
		(*svs)[secVarType][nsName] = make(map[string]SecVar)
	}

	// Assign the value
	(*svs)[secVarType][nsName][ghPropertyName] = secVar
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

//
//
//

// TODO: handle timeouts, requeue with "return ctrl.Result{RequeueAfter: time.Minute}, nil" ?
func SynchronizeToGithub(ctx context.Context, cli client.Client, ghCli github.Client, toApplyTo []*qalisav1alpha1.GithubSyncRepo, secVarsToSync SecVarsBySync) (ctrl.Result, error) {
	//
	//
	//
	var ghPropsSyncStateDict *[]qalisav1alpha1.GithubPropertySyncState

	//
	//
	// for each repository to sync...
	for _, repoCRD := range toApplyTo {
		//
		var secAttempts SyncAttempts
		var varAttempts SyncAttempts
		var resultStatsStr string

		//
		// Try to parse repo
		//
		repo, err := ParseRepository(*repoCRD)
		if err != nil {
			SetSyncedStatusCondition(repoCRD, &repoCRD.Status.Conditions, "False", err.Error())
			// if failed, skip syncing altogether
			goto doRegisterStatus
		}

		//
		//
		// handle SECRETS bucket...
		ghPropsSyncStateDict = &repoCRD.Status.SecretsSyncStates
		for _, secretsBucket := range secVarsToSync[Secret] {
			// for each secret...
			for githubSecretName, secVar := range secretsBucket {
				secAttempts.BumpTotal()

				// try to find if already synced
				if isGHPropertyAlreadySynced(ghPropsSyncStateDict, githubSecretName, secVar) {
					secAttempts.BumpNotNeeded()
					continue
				}

				// if not, try to update w/ Github API
				err := secVar.UpdateAgainstGithubApiAs(ctx, ghCli, Secret, repo, githubSecretName)

				// whatever the result, define sync state
				defineGHPropertySyncStatus(repoCRD, ghPropsSyncStateDict, githubSecretName, secVar, err, &secAttempts)
			}
		}

		//
		//
		// handle VARIABLES bucket...
		ghPropsSyncStateDict = &repoCRD.Status.VariablesSyncStates
		for _, variableBucket := range secVarsToSync[Variable] {
			// for each variable...
			for githubVariableName, secVar := range variableBucket {
				varAttempts.BumpTotal()

				// try to find if already synced
				if isGHPropertyAlreadySynced(ghPropsSyncStateDict, githubVariableName, secVar) {
					secAttempts.BumpNotNeeded()
					continue
				}

				// if not, try to update w/ Github API
				err := secVar.UpdateAgainstGithubApiAs(ctx, ghCli, Variable, repo, githubVariableName)

				// whatever the result, define sync state
				defineGHPropertySyncStatus(repoCRD, ghPropsSyncStateDict, githubVariableName, secVar, err, &varAttempts)
			}
		}

		//
		//
		//

		//
		resultStatsStr = fmt.Sprintf(
			"(Synced: %d/%d variables | %d/%d secrets)",
			varAttempts.DoneOrDidSuccess(), varAttempts.Total(),
			secAttempts.DoneOrDidSuccess(), secAttempts.Total(),
		)

		if varAttempts.HasFailed() || secAttempts.HasFailed() {
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
			// Kind of anormal error; Would immediately schedule requeue because of err is set
			return ctrl.Result{}, err
		}
	}

	//
	return ctrl.Result{}, nil
}
