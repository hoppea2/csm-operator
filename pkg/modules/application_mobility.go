//  Copyright © 2023 Dell Inc. or its subsidiaries. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//       http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package modules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	csmv1 "github.com/dell/csm-operator/api/v1"
	"github.com/dell/csm-operator/pkg/logger"
	operatorutils "github.com/dell/csm-operator/pkg/operatorutils"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (

	// AppMobDeploymentManifest - filename of deployment manifest for app-mobility
	AppMobDeploymentManifest = "app-mobility-controller-manager.yaml"
	// AppMobMetricService - filename of MetricService manifest for app-mobility
	AppMobMetricService = "app-mobility-controller-manager-metrics-service.yaml"
	// AppMobWebhookService - filename of Webhook manifest for app-mobility
	AppMobWebhookService = "app-mobility-webhook-service.yaml"
	// AppMobCrds - name of app-mobility crd manifest yaml
	AppMobCrds = "app-mobility-crds.yaml"
	// VeleroManifest - filename of Velero manifest for app-mobility
	VeleroManifest = "velero-deployment.yaml"
	// AppMobCertManagerManifest - filename of Cert-manager manifest for app-mobility
	AppMobCertManagerManifest = "cert-manager.yaml"
	// UseVolSnapshotManifest - filename of use volume snapshot manifest for velero
	UseVolSnapshotManifest = "velero-volumesnapshotlocation.yaml"
	// BackupStorageLoc - filename of backupstoragelocation manifest for velero
	BackupStorageLoc = "velero-backupstoragelocation.yaml"
	// CleanupCrdManifest - filename of Cleanup Crds manifest for app-mobility
	CleanupCrdManifest = "cleanupcrds.yaml"
	// VeleroCrdManifest - filename of Velero crds manisfest for Velero feature
	VeleroCrdManifest = "velero-crds.yaml"
	// VeleroAccessManifest - filename where velero access with its contents
	VeleroAccessManifest = "velero-secret.yaml"
	// CertManagerIssuerCertManifest - filename of the issuer and cert for app-mobility
	CertManagerIssuerCertManifest = "certificate.yaml"
	// NodeAgentCrdManifest - filename of node-agent manifest for app-mobility
	NodeAgentCrdManifest = "node-agent.yaml"

	// ControllerImg - image for app-mobility-controller
	ControllerImg = "<CONTROLLER_IMAGE>"
	// ControllerImagePullPolicy - default image pull policy in yamls
	ControllerImagePullPolicy = "<CONTROLLER_IMAGE_PULLPOLICY>"
	// AppMobNamespace - namespace Application Mobility is installed in
	AppMobNamespace = "<NAMESPACE>"
	// AppMobReplicaCount - Number of replicas
	AppMobReplicaCount = "<APPLICATION_MOBILITY_REPLICA_COUNT>"
	// AppMobObjStoreSecretName - Secret name for object store
	AppMobObjStoreSecretName = "<APPLICATION_MOBILITY_OBJECT_STORE_SECRET_NAME>"
	// BackupStorageLocation - name for Backup Storage Location
	BackupStorageLocation = "<BACKUPSTORAGELOCATION_NAME>"
	// VeleroBucketName - name for the used velero bucket
	VeleroBucketName = "<BUCKET_NAME>"
	// VeleroCaCert - name for the used velero cacert
	VeleroCaCert = "<BUCKET_CACERT>"
	// VolSnapshotlocation - name for Volume Snapshot location
	VolSnapshotlocation = "<VOL_SNAPSHOT_LOCATION_NAME>"
	// BackupStorageURL - cloud url for backup storage location
	BackupStorageURL = "<BACKUP_STORAGE_URL>"
	// BackupStorageRegion - region for backup to take place in
	BackupStorageRegion = "<BACKUP_REGION_URL>"
	// BackupStorageRegionDefault - default value if BACKUP_REGION_URL is not specified
	BackupStorageRegionDefault = "region"
	// ConfigProvider - configurations provider
	ConfigProvider = "<CONFIGURATION_PROVIDER>"
	// VeleroImage - Image for velero
	VeleroImage = "<VELERO_IMAGE>"
	// VeleroImagePullPolicy - image pull policy for velero
	VeleroImagePullPolicy = "<VELERO_IMAGE_PULLPOLICY>"
	// VeleroAccess  -  Secret name for velero
	VeleroAccess = "<VELERO_ACCESS>"
	// AWSInitContainerName - Name of init container for velero - aws
	AWSInitContainerName = "<AWS_INIT_CONTAINER_NAME>"
	// AWSInitContainerImage - Image of init container for velero -aws
	AWSInitContainerImage = "<AWS_INIT_CONTAINER_IMAGE>"
	// DELLInitContainerName - Name of init container for velero - dell
	DELLInitContainerName = "<DELL_INIT_CONTAINER_NAME>"
	// DELLInitContainerImage - Image of init container for velero - dell
	DELLInitContainerImage = "<DELL_INIT_CONTAINER_IMAGE>"
	// AccessContents - contents of the object store secret
	AccessContents = "<CRED_CONTENTS>"
	// AKeyID - contains the aws access key id
	AKeyID = "<KEY_ID>"
	// AKey - contains the aws access key
	AKey = "<KEY>"

	// AppMobCtrlMgrComponent - component name in cr for app-mobility controller-manager
	AppMobCtrlMgrComponent = "application-mobility-controller-manager"
	// AppMobCertManagerComponent - cert-manager component
	AppMobCertManagerComponent = "cert-manager"
	// AppMobVeleroComponent - velero component
	AppMobVeleroComponent = "velero"

	// AppMobilityCSMNameSpace - namespace CSM is found in. Needed for cases where pod namespace is not namespace of CSM
	AppMobilityCSMNameSpace string = "<CSM_NAMESPACE>"
)

// ApplicationMobilityOldVersion - old version of application-mobility, will be filled in checkUpgrade
var ApplicationMobilityOldVersion = ""

// getAppMobilityModule - get instance of app mobility module
func getAppMobilityModule(cr csmv1.ContainerStorageModule) (csmv1.Module, error) {
	for _, m := range cr.Spec.Modules {
		if m.Name == csmv1.ApplicationMobility {
			return m, nil
		}
	}
	return csmv1.Module{}, fmt.Errorf("Application Mobility module not found")
}

// getVeleroCrdDeploy - applies and deploy VeleroCrd manifest
func getVeleroCrdDeploy(op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule) (string, error) {
	yamlString := ""

	appMob, err := getAppMobilityModule(cr)
	if err != nil {
		return yamlString, err
	}

	veleroCrdPath := fmt.Sprintf("%s/moduleconfig/application-mobility/%s/%s", op.ConfigDirectory, appMob.ConfigVersion, VeleroCrdManifest)
	buf, err := os.ReadFile(filepath.Clean(veleroCrdPath))
	if err != nil {
		return yamlString, err
	}

	yamlString = string(buf)

	return yamlString, nil
}

// VeleroCrdDeploy - apply and delete Velero crds deployment
func VeleroCrdDeploy(ctx context.Context, op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule, ctrlClient crclient.Client) error {
	yamlString, err := getVeleroCrdDeploy(op, cr)
	if err != nil {
		return err
	}

	ctrlObjects, err := operatorutils.GetModuleComponentObj([]byte(yamlString))
	if err != nil {
		return err
	}

	for _, ctrlObj := range ctrlObjects {
		if err := operatorutils.ApplyObject(ctx, ctrlObj, ctrlClient); err != nil {
			return err
		}
	}

	return nil
}

// getAppMobCrdDeploy - apply and deploy app mobility crd manifest
func getAppMobCrdDeploy(op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule) (string, error) {
	yamlString := ""

	appMob, err := getAppMobilityModule(cr)
	if err != nil {
		return yamlString, err
	}

	appMobCrdPath := fmt.Sprintf("%s/moduleconfig/application-mobility/%s/%s", op.ConfigDirectory, appMob.ConfigVersion, AppMobCrds)
	buf, err := os.ReadFile(filepath.Clean(appMobCrdPath))
	if err != nil {
		return yamlString, err
	}

	yamlString = string(buf)

	yamlString = strings.ReplaceAll(yamlString, AppMobNamespace, cr.Namespace)
	yamlString = strings.ReplaceAll(yamlString, AppMobilityCSMNameSpace, cr.Namespace)

	return yamlString, nil
}

// AppMobCrdDeploy - apply and delete Velero crds deployment
func AppMobCrdDeploy(ctx context.Context, op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule, ctrlClient crclient.Client) error {
	yamlString, err := getAppMobCrdDeploy(op, cr)
	if err != nil {
		return err
	}

	ctrlObjects, err := operatorutils.GetModuleComponentObj([]byte(yamlString))
	if err != nil {
		return err
	}

	for _, ctrlObj := range ctrlObjects {
		if err := operatorutils.ApplyObject(ctx, ctrlObj, ctrlClient); err != nil {
			return err
		}
	}

	return nil
}

// getAppMobilityModuleDeployment - updates deployment manifest with app mobility CRD values
func getAppMobilityModuleDeployment(op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule) (string, error) {
	yamlString := ""
	appMob, err := getAppMobilityModule(cr)
	if err != nil {
		return yamlString, err
	}

	deploymentPath := fmt.Sprintf("%s/moduleconfig/application-mobility/%s/%s", op.ConfigDirectory, appMob.ConfigVersion, AppMobDeploymentManifest)
	buf, err := os.ReadFile(filepath.Clean(deploymentPath))
	if err != nil {
		return yamlString, err
	}

	yamlString = string(buf)
	controllerImage := ""
	controllerImagePullPolicy := ""
	replicaCount := ""
	objectSecretName := ""

	for _, component := range appMob.Components {
		if component.Name == AppMobCtrlMgrComponent {
			controllerImage = string(component.Image)
			controllerImagePullPolicy = string(component.ImagePullPolicy)
			for _, env := range component.Envs {
				if strings.Contains(AppMobReplicaCount, env.Name) {
					replicaCount = env.Value
				}
			}
		}
		if component.Name == AppMobVeleroComponent {
			for _, env := range component.Envs {
				if strings.Contains(AppMobObjStoreSecretName, env.Name) {
					objectSecretName = env.Value
				}
			}
		}
		for _, cred := range component.ComponentCred {
			if cred.CreateWithInstall {
				yamlString = strings.ReplaceAll(yamlString, AppMobObjStoreSecretName, cred.Name)
			} else {
				yamlString = strings.ReplaceAll(yamlString, AppMobObjStoreSecretName, objectSecretName)
			}
		}
	}

	yamlString = strings.ReplaceAll(yamlString, CSMName, cr.Name)
	yamlString = strings.ReplaceAll(yamlString, AppMobNamespace, cr.Namespace)
	yamlString = strings.ReplaceAll(yamlString, ControllerImg, controllerImage)
	yamlString = strings.ReplaceAll(yamlString, ControllerImagePullPolicy, controllerImagePullPolicy)
	yamlString = strings.ReplaceAll(yamlString, AppMobReplicaCount, replicaCount)
	yamlString = strings.ReplaceAll(yamlString, AppMobilityCSMNameSpace, cr.Namespace)

	return yamlString, nil
}

// AppMobilityDeployment - apply and delete controller manager deployment
func AppMobilityDeployment(ctx context.Context, isDeleting bool, op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule, ctrlClient crclient.Client) error {
	yamlString, err := getAppMobilityModuleDeployment(op, cr)
	if err != nil {
		return err
	}

	er := applyDeleteObjects(ctx, ctrlClient, yamlString, isDeleting)
	if er != nil {
		return er
	}

	return nil
}

// getControllerManagerMetricService - updates metric manifest with app mobility CRD values
func getControllerManagerMetricService(op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule) (string, error) {
	yamlString := ""

	appMob, err := getAppMobilityModule(cr)
	if err != nil {
		return yamlString, err
	}

	metricServicePath := fmt.Sprintf("%s/moduleconfig/application-mobility/%s/%s", op.ConfigDirectory, appMob.ConfigVersion, AppMobMetricService)
	buf, err := os.ReadFile(filepath.Clean(metricServicePath))
	if err != nil {
		return yamlString, err
	}

	yamlString = string(buf)
	yamlString = strings.ReplaceAll(yamlString, AppMobNamespace, cr.Namespace)

	return yamlString, nil
}

// ControllerManagerMetricService - apply and delete Controller manager metric service deployment
func ControllerManagerMetricService(ctx context.Context, isDeleting bool, op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule, ctrlClient crclient.Client) error {
	yamlString, err := getControllerManagerMetricService(op, cr)
	if err != nil {
		return err
	}

	er := applyDeleteObjects(ctx, ctrlClient, yamlString, isDeleting)
	if er != nil {
		return er
	}

	return nil
}

// getAppMobilityWebhookService - gets the app mobility webhook service manifest
func getAppMobilityWebhookService(op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule) (string, error) {
	yamlString := ""
	appMob, err := getAppMobilityModule(cr)
	if err != nil {
		return yamlString, err
	}

	webhookServicePath := fmt.Sprintf("%s/moduleconfig/application-mobility/%s/%s", op.ConfigDirectory, appMob.ConfigVersion, AppMobWebhookService)
	buf, err := os.ReadFile(filepath.Clean(webhookServicePath))
	if err != nil {
		return yamlString, err
	}

	yamlString = string(buf)
	yamlString = strings.ReplaceAll(yamlString, AppMobNamespace, cr.Namespace)

	return yamlString, nil
}

// AppMobilityWebhookService - apply/delete app mobility's webhook service
func AppMobilityWebhookService(ctx context.Context, isDeleting bool, op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule, ctrlClient crclient.Client) error {
	yamlString, err := getAppMobilityWebhookService(op, cr)
	if err != nil {
		return err
	}

	er := applyDeleteObjects(ctx, ctrlClient, yamlString, isDeleting)
	if er != nil {
		return er
	}

	return nil
}

// getIssuerCertService - gets the app mobility cert manager's issuer and certificate manifest
func getIssuerCertService(op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule) (string, error) {
	yamlString := ""
	appMob, err := getAppMobilityModule(cr)
	if err != nil {
		return yamlString, err
	}

	issuerCertServicePath := fmt.Sprintf("%s/moduleconfig/application-mobility/%s/%s", op.ConfigDirectory, appMob.ConfigVersion, CertManagerIssuerCertManifest)
	buf, err := os.ReadFile(filepath.Clean(issuerCertServicePath))
	if err != nil {
		return yamlString, err
	}

	yamlString = string(buf)
	yamlString = strings.ReplaceAll(yamlString, AppMobNamespace, cr.Namespace)

	return yamlString, nil
}

// IssuerCertService - apply and delete the app mobility issuer and certificate service
func IssuerCertService(ctx context.Context, isDeleting bool, op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule, ctrlClient crclient.Client) error {
	yamlString, err := getIssuerCertService(op, cr)
	if err != nil {
		return err
	}

	er := applyDeleteObjects(ctx, ctrlClient, yamlString, isDeleting)
	if er != nil {
		return er
	}

	return nil
}

// ApplicationMobilityPrecheck - runs precheck for CSM Application Mobility
func ApplicationMobilityPrecheck(ctx context.Context, op operatorutils.OperatorConfig, appMob csmv1.Module, _ csmv1.ContainerStorageModule, r operatorutils.ReconcileCSM) error {
	log := logger.GetLogger(ctx)

	// check if provided version is supported
	if appMob.ConfigVersion != "" {
		err := checkVersion(string(csmv1.ApplicationMobility), appMob.ConfigVersion, op.ConfigDirectory)
		if err != nil {
			return err
		}
	}

	// Check for secrets
	ns := "default"
	appMobilitySecrets := []string{"dls-license", "iv"}
	for _, name := range appMobilitySecrets {
		found := &corev1.Secret{}
		err := r.GetClient().Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, found)
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return fmt.Errorf("failed to find secret %s", name)
			}
		}
	}

	log.Infof("performed pre-checks for %s", appMob.Name)
	return nil
}

// AppMobilityCertManager - Install/Delete cert-manager
func AppMobilityCertManager(ctx context.Context, isDeleting bool, op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule, ctrlClient crclient.Client) error {
	yamlString, err := getCertManager(op, cr)
	if err != nil {
		return err
	}

	er := applyDeleteObjects(ctx, ctrlClient, yamlString, isDeleting)
	if er != nil {
		return er
	}

	return nil
}

// CreateVeleroAccess - Install/Delete velero-secret yaml from operator config
func CreateVeleroAccess(ctx context.Context, isDeleting bool, op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule, ctrlClient crclient.Client) error {
	yamlString, err := getCreateVeleroAccess(op, cr)
	if err != nil {
		return err
	}

	er := applyDeleteObjects(ctx, ctrlClient, yamlString, isDeleting)
	if er != nil {
		return er
	}

	return nil
}

// getCreateVeleroAccess - gets the velero-secret manifest from operatorconfig
func getCreateVeleroAccess(op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule) (string, error) {
	yamlString := ""

	appMob, err := getAppMobilityModule(cr)
	if err != nil {
		return yamlString, err
	}

	veleroAccessPath := fmt.Sprintf("%s/moduleconfig/application-mobility/%s/%s", op.ConfigDirectory, appMob.ConfigVersion, VeleroAccessManifest)
	buf, err := os.ReadFile(filepath.Clean(veleroAccessPath))
	if err != nil {
		return yamlString, err
	}

	yamlString = string(buf)
	credName := ""
	backupStorageLocationName := ""
	accessID := ""
	access := ""

	for _, component := range appMob.Components {
		if component.Name == AppMobVeleroComponent {
			for _, env := range component.Envs {
				if strings.Contains(BackupStorageLocation, env.Name) {
					backupStorageLocationName = env.Value
				}
			}
		}
		for _, cred := range component.ComponentCred {
			if cred.CreateWithInstall {
				credName = string(cred.Name)
				accessID = string(cred.SecretContents.AccessKeyID)
				access = string(cred.SecretContents.AccessKey)

			}
		}

	}

	yamlString = strings.ReplaceAll(yamlString, AppMobNamespace, cr.Namespace)
	yamlString = strings.ReplaceAll(yamlString, VeleroAccess, credName)
	yamlString = strings.ReplaceAll(yamlString, BackupStorageLocation, backupStorageLocationName)
	yamlString = strings.ReplaceAll(yamlString, AKeyID, accessID)
	yamlString = strings.ReplaceAll(yamlString, AKey, access)
	yamlString = strings.ReplaceAll(yamlString, AppMobilityCSMNameSpace, cr.Namespace)

	return yamlString, nil
}

// AppMobilityVelero - Install/Delete velero along with its features - use volume snapshot location and cleanup crds
func AppMobilityVelero(ctx context.Context, isDeleting bool, op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule, ctrlClient crclient.Client) error {
	var useSnap bool
	var nodeAgent bool
	envCredName := ""
	compCredName := ""
	log := logger.GetLogger(ctx)

	yamlString, err := getVelero(op, cr)
	if err != nil {
		return err
	}

	er := applyDeleteObjects(ctx, ctrlClient, yamlString, isDeleting)
	if er != nil {
		return er
	}

	for _, m := range cr.Spec.Modules {
		if m.Name == csmv1.ApplicationMobility {
			for _, c := range m.Components {
				if c.Name == AppMobVeleroComponent {
					if c.UseSnapshot {
						useSnap = true
					}
					if c.DeployNodeAgent {
						nodeAgent = true
					}
					for _, env := range c.Envs {
						if strings.Contains(AppMobObjStoreSecretName, env.Name) {
							envCredName = env.Value
						}
					}
					for _, cred := range c.ComponentCred {
						// if createWithInstall is enabled then create a secret
						if cred.CreateWithInstall {
							compCredName = string(cred.Name)
							foundCred, _ := operatorutils.GetSecret(ctx, compCredName, cr.Namespace, ctrlClient)
							if foundCred == nil {
								// creation of a secret
								err := CreateVeleroAccess(ctx, isDeleting, op, cr, ctrlClient)
								if err != nil {
									return fmt.Errorf("\n Unable to deploy velero-secret for Application Mobility: %v", err)
								}
							}
						} else {
							foundCred, err := operatorutils.GetSecret(ctx, envCredName, cr.Namespace, ctrlClient)
							if foundCred == nil {
								log.Errorw("\n The secret : %s ", envCredName, " cannot be found in the provided namespace")
								return fmt.Errorf("\n Unable to deploy velero-secret for Application Mobility: %v", err)
							}
						}
					}
				}
			}
		}
	}

	// create volume snapshot location
	if useSnap {

		vsName, yamlString2, err := getUseVolumeSnapshot(ctx, op, cr, ctrlClient)
		if err != nil {
			return err
		}

		volumeSnapshotLoc, _ := operatorutils.GetVolumeSnapshotLocation(ctx, vsName, cr.Namespace, ctrlClient)
		if volumeSnapshotLoc != nil {
			log.Infow("\n Volume Snapshot location Name : ", volumeSnapshotLoc.Name, " already exists and being re-used")
		}

		ctrlObjects, err := operatorutils.GetModuleComponentObj([]byte(yamlString2))
		if err != nil {
			return err
		}

		for _, ctrlObj := range ctrlObjects {
			if !isDeleting {
				if err := operatorutils.ApplyObject(ctx, ctrlObj, ctrlClient); err != nil {
					return err
				}
			}
		}
	}

	// enable node agent
	if nodeAgent {
		yamlString4, err := getNodeAgent(op, cr)
		if err != nil {
			return err
		}

		newVersion := cr.GetModule(csmv1.ApplicationMobility).ConfigVersion

		// if moving AM versions, need to remove old node agent Daemonset due to name change
		if newVersion != ApplicationMobilityOldVersion && ApplicationMobilityOldVersion != "" {
			log.Infow("Need to remove old node agent Daemonset")
			if err := RemoveOldDaemonset(ctx, op, ApplicationMobilityOldVersion, cr, ctrlClient); err != nil {
				log.Warnf("Failed to remove old node agent Daemonset: %s", err)
			}
		}

		er := applyDeleteObjects(ctx, ctrlClient, yamlString4, isDeleting)
		if er != nil {
			return er
		}
	}
	return nil
}

// getVelero - gets the velero-deployment manifest
func getVelero(op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule) (string, error) {
	yamlString := ""

	appMob, err := getAppMobilityModule(cr)
	if err != nil {
		return yamlString, err
	}

	veleroPath := fmt.Sprintf("%s/moduleconfig/application-mobility/%s/%s", op.ConfigDirectory, appMob.ConfigVersion, VeleroManifest)
	buf, err := os.ReadFile(filepath.Clean(veleroPath))
	if err != nil {
		return yamlString, err
	}

	yamlString = string(buf)
	veleroImg := ""
	veleroImgPullPolicy := ""
	veleroAWSInitContainerName := ""
	veleroAWSInitContainerImage := ""
	veleroDELLInitContainerName := ""
	veleroDELLInitContainerImage := ""
	objectSecretName := ""

	for _, component := range appMob.Components {
		if component.Name == AppMobVeleroComponent {
			if component.Image != "" {
				veleroImg = string(component.Image)
			}
			if component.ImagePullPolicy != "" {
				veleroImgPullPolicy = string(component.ImagePullPolicy)
			}
			for _, env := range component.Envs {
				if strings.Contains(AppMobObjStoreSecretName, env.Name) {
					objectSecretName = env.Value
				}
			}
			for _, cred := range component.ComponentCred {
				if cred.CreateWithInstall {
					yamlString = strings.ReplaceAll(yamlString, AppMobObjStoreSecretName, cred.Name)
				} else {
					yamlString = strings.ReplaceAll(yamlString, AppMobObjStoreSecretName, objectSecretName)
				}
			}
		}
	}
	for _, m := range cr.Spec.Modules {
		for _, icontainer := range m.InitContainer {
			if icontainer.Name == "velero-plugin-for-aws" {
				veleroAWSInitContainerName = icontainer.Name
				veleroAWSInitContainerImage = string(icontainer.Image)
			}
			if icontainer.Name == "dell-custom-velero-plugin" {
				veleroDELLInitContainerName = icontainer.Name
				veleroDELLInitContainerImage = string(icontainer.Image)
			}
		}
	}

	yamlString = strings.ReplaceAll(yamlString, CSMName, cr.Name)
	yamlString = strings.ReplaceAll(yamlString, AppMobNamespace, cr.Namespace)
	yamlString = strings.ReplaceAll(yamlString, VeleroImage, veleroImg)
	yamlString = strings.ReplaceAll(yamlString, VeleroImagePullPolicy, veleroImgPullPolicy)
	yamlString = strings.ReplaceAll(yamlString, AWSInitContainerName, veleroAWSInitContainerName)
	yamlString = strings.ReplaceAll(yamlString, AWSInitContainerImage, veleroAWSInitContainerImage)
	yamlString = strings.ReplaceAll(yamlString, DELLInitContainerName, veleroDELLInitContainerName)
	yamlString = strings.ReplaceAll(yamlString, DELLInitContainerImage, veleroDELLInitContainerImage)
	yamlString = strings.ReplaceAll(yamlString, AppMobilityCSMNameSpace, cr.Namespace)

	return yamlString, nil
}

// getUseVolumeSnapshot - gets the velero - volume snapshot location manifest
func getUseVolumeSnapshot(_ context.Context, op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule, _ crclient.Client) (string, string, error) {
	yamlString := ""

	appMob, err := getAppMobilityModule(cr)
	if err != nil {
		return "Error: ", yamlString, err
	}

	volSnapshotPath := fmt.Sprintf("%s/moduleconfig/application-mobility/%s/%s", op.ConfigDirectory, appMob.ConfigVersion, UseVolSnapshotManifest)
	buf, err := os.ReadFile(filepath.Clean(volSnapshotPath))
	if err != nil {
		return "Error: ", yamlString, err
	}

	yamlString = string(buf)
	volSnapshotLocationName := ""
	provider := ""
	backupRegion := ""
	for _, component := range appMob.Components {
		if component.Name == AppMobVeleroComponent {
			for _, env := range component.Envs {
				if strings.Contains(VolSnapshotlocation, env.Name) {
					volSnapshotLocationName = env.Value
				}
				if strings.Contains(ConfigProvider, env.Name) {
					provider = env.Value
				}
				if strings.Contains(BackupStorageRegion, env.Name) {
					backupRegion = env.Value
				}
			}
		}
	}

	// if BackupStorageRegion is not provided - use default variable
	if backupRegion == "" {
		backupRegion = BackupStorageRegionDefault
	}

	yamlString = strings.ReplaceAll(yamlString, AppMobNamespace, cr.Namespace)
	yamlString = strings.ReplaceAll(yamlString, VolSnapshotlocation, volSnapshotLocationName)
	yamlString = strings.ReplaceAll(yamlString, ConfigProvider, provider)
	yamlString = strings.ReplaceAll(yamlString, BackupStorageRegion, backupRegion)
	yamlString = strings.ReplaceAll(yamlString, AppMobilityCSMNameSpace, cr.Namespace)

	return volSnapshotLocationName, yamlString, nil
}

// getBackupStorageLoc - gets the velero Backup Storage Location manifest
func getBackupStorageLoc(_ context.Context, op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule, _ crclient.Client) (string, string, error) {
	yamlString := ""

	appMob, err := getAppMobilityModule(cr)
	if err != nil {
		return "Error: ", yamlString, err
	}

	BackupStorageLocPath := fmt.Sprintf("%s/moduleconfig/application-mobility/%s/%s", op.ConfigDirectory, appMob.ConfigVersion, BackupStorageLoc)
	buf, err := os.ReadFile(filepath.Clean(BackupStorageLocPath))
	if err != nil {
		return "Error: ", yamlString, err
	}

	yamlString = string(buf)
	backupStorageLocationName := ""
	provider := ""
	bucketName := ""
	backupURL := ""
	backupRegion := ""
	bucketCacert := ""
	for _, component := range appMob.Components {
		if component.Name == AppMobVeleroComponent {
			for _, env := range component.Envs {
				if strings.Contains(BackupStorageLocation, env.Name) {
					backupStorageLocationName = env.Value
				}
				if strings.Contains(VeleroBucketName, env.Name) {
					bucketName = env.Value
				}
				if strings.Contains(ConfigProvider, env.Name) {
					provider = env.Value
				}
				if strings.Contains(BackupStorageURL, env.Name) {
					backupURL = env.Value
				}
				if strings.Contains(BackupStorageRegion, env.Name) {
					backupRegion = env.Value
				}
				if strings.Contains(VeleroCaCert, env.Name) {
					bucketCacert = env.Value
				}
			}
		}
	}

	// if BackupStorageRegion is not provided - use default variable
	if backupRegion == "" {
		backupRegion = BackupStorageRegionDefault
	}

	yamlString = strings.ReplaceAll(yamlString, AppMobNamespace, cr.Namespace)
	yamlString = strings.ReplaceAll(yamlString, BackupStorageLocation, backupStorageLocationName)
	yamlString = strings.ReplaceAll(yamlString, VeleroBucketName, bucketName)
	yamlString = strings.ReplaceAll(yamlString, BackupStorageURL, backupURL)
	yamlString = strings.ReplaceAll(yamlString, ConfigProvider, provider)
	yamlString = strings.ReplaceAll(yamlString, BackupStorageRegion, backupRegion)
	yamlString = strings.ReplaceAll(yamlString, VeleroCaCert, bucketCacert)
	yamlString = strings.ReplaceAll(yamlString, AppMobilityCSMNameSpace, cr.Namespace)

	return backupStorageLocationName, yamlString, nil
}

// UseBackupStorageLoc - Apply/Delete velero-backupstoragelocation yaml from operator config
func UseBackupStorageLoc(ctx context.Context, isDeleting bool, op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule, ctrlClient crclient.Client) error {
	log := logger.GetLogger(ctx)
	bslName, yamlString, err := getBackupStorageLoc(ctx, op, cr, ctrlClient)
	if err != nil {
		return err
	}

	backupStorageLoc, _ := operatorutils.GetBackupStorageLocation(ctx, bslName, cr.Namespace, ctrlClient)
	if backupStorageLoc != nil {
		log.Infow("\n Backup Storage Name : ", backupStorageLoc.Name, "already exists and being re-used")
	}

	ctrlObjects, err := operatorutils.GetModuleComponentObj([]byte(yamlString))
	if err != nil {
		return err
	}

	for _, ctrlObj := range ctrlObjects {
		if !isDeleting {
			if err := operatorutils.ApplyObject(ctx, ctrlObj, ctrlClient); err != nil {
				return err
			}
		}
	}

	return nil
}

// getNodeAgent - gets node-agent services manifests
func getNodeAgent(op operatorutils.OperatorConfig, cr csmv1.ContainerStorageModule) (string, error) {
	yamlString := ""

	appMob, err := getAppMobilityModule(cr)
	if err != nil {
		return yamlString, err
	}
	cleanupCrdsPath := fmt.Sprintf("%s/moduleconfig/application-mobility/%s/%s", op.ConfigDirectory, appMob.ConfigVersion, NodeAgentCrdManifest)
	buf, err := os.ReadFile(filepath.Clean(cleanupCrdsPath))
	if err != nil {
		return yamlString, err
	}

	yamlString = string(buf)
	veleroImgPullPolicy := ""
	veleroImg := ""
	objectSecretName := ""

	for _, component := range appMob.Components {
		if component.Name == AppMobVeleroComponent {
			if component.Image != "" {
				veleroImg = string(component.Image)
			}
			if component.ImagePullPolicy != "" {
				veleroImgPullPolicy = string(component.ImagePullPolicy)
			}
			for _, env := range component.Envs {
				if strings.Contains(AppMobObjStoreSecretName, env.Name) {
					objectSecretName = env.Value
				}
			}
			for _, cred := range component.ComponentCred {
				if cred.CreateWithInstall {
					yamlString = strings.ReplaceAll(yamlString, AppMobObjStoreSecretName, cred.Name)
				} else {
					yamlString = strings.ReplaceAll(yamlString, AppMobObjStoreSecretName, objectSecretName)
				}
			}
		}
	}

	yamlString = strings.ReplaceAll(yamlString, VeleroImage, veleroImg)
	yamlString = strings.ReplaceAll(yamlString, AppMobNamespace, cr.Namespace)
	yamlString = strings.ReplaceAll(yamlString, VeleroImagePullPolicy, veleroImgPullPolicy)
	yamlString = strings.ReplaceAll(yamlString, AppMobilityCSMNameSpace, cr.Namespace)
	return yamlString, nil
}

// applyDeleteObjects - Applies/Deletes the object based on boolean value
func applyDeleteObjects(ctx context.Context, ctrlClient crclient.Client, yamlString string, isDeleting bool) error {
	ctrlObjects, err := operatorutils.GetModuleComponentObj([]byte(yamlString))
	if err != nil {
		return err
	}

	for _, ctrlObj := range ctrlObjects {
		if isDeleting {
			if err := operatorutils.DeleteObject(ctx, ctrlObj, ctrlClient); err != nil {
				return err
			}
		} else {
			if err := operatorutils.ApplyObject(ctx, ctrlObj, ctrlClient); err != nil {
				return err
			}
		}
	}

	return nil
}

// RemoveOldDaemonset is used to remove Daemonset if switching between AM versions
func RemoveOldDaemonset(ctx context.Context, op operatorutils.OperatorConfig, oldVersion string, cr csmv1.ContainerStorageModule, ctrlClient crclient.Client) error {
	log := logger.GetLogger(ctx)
	// need to delete the old Daemonset, which is found in versions v1.0.3 or lower
	log.Infof("removing application-mobility-node-agent daemonset from %s namespace", cr.Namespace)
	oldNodeAgentPath := fmt.Sprintf("%s/moduleconfig/application-mobility/%s/%s", op.ConfigDirectory, oldVersion, NodeAgentCrdManifest)

	buf, err := os.ReadFile(filepath.Clean(oldNodeAgentPath))
	if err != nil {
		return fmt.Errorf("failed to find read old node-agent manifests at path: %s", oldNodeAgentPath)
	}
	yamlString := string(buf)
	yamlString = strings.ReplaceAll(yamlString, AppMobNamespace, cr.Namespace)
	yamlString = strings.ReplaceAll(yamlString, AppMobilityCSMNameSpace, cr.Namespace)
	return applyDeleteObjects(context.Background(), ctrlClient, yamlString, true)
}
