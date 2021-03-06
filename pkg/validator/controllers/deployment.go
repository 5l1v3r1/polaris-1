package controllers

import (
	"github.com/fairwindsops/polaris/pkg/config"
	kubeAPIAppsV1 "k8s.io/api/apps/v1"
)

// NewDeploymentController builds a new controller interface for Deployments
func NewDeploymentController(originalResource kubeAPIAppsV1.Deployment) GenericController {
	controller := GenericController{}
	controller.Name = originalResource.Name
	controller.Namespace = originalResource.Namespace
	controller.PodSpec = originalResource.Spec.Template.Spec
	controller.ObjectMeta = originalResource.ObjectMeta
	controller.Kind = config.Deployments.String()
	return controller
}
