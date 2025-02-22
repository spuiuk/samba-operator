/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources

import (
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/samba-in-kubernetes/samba-operator/internal/conf"
	pln "github.com/samba-in-kubernetes/samba-operator/internal/planner"
)

var (
	serviceLabel = "samba-operator.samba.org/service"
)

// buildDeployment returns a samba server deployment object
func buildDeployment(cfg *conf.OperatorConfig,
	planner *pln.Planner, pvcName, ns string) *appsv1.Deployment {
	// construct a deployment based on the following labels
	labels := labelsForSmbServer(planner.InstanceName())
	var size int32 = 1

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      planner.InstanceName(),
			Namespace: ns,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotationsForSmbPod(cfg),
				},
				Spec: buildPodSpec(planner, cfg, pvcName),
			},
		},
	}
	return deployment
}

// labelsForSmbServer returns the labels for selecting the resources
// belonging to the given CR name.
func labelsForSmbServer(name string) map[string]string {
	return map[string]string{
		// top level labes
		"app": "samba",
		// k8s namespaced labels
		// See: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
		"app.kubernetes.io/name":       "samba",
		"app.kubernetes.io/instance":   labelValue("samba", name),
		"app.kubernetes.io/component":  "smbd",
		"app.kubernetes.io/part-of":    "samba",
		"app.kubernetes.io/managed-by": "samba-operator",
		// our namespaced labels
		serviceLabel: labelValue(name),
	}
}

func labelValue(s ...string) string {
	out := strings.Join(s, "-")
	if len(out) > 63 {
		out = out[:63]
	}
	return out
}

func annotationsForSmbPod(cfg *conf.OperatorConfig) map[string]string {
	name := cfg.SmbdContainerName
	annotations := map[string]string{
		"kubectl.kubernetes.io/default-logs-container": name,
		"kubectl.kubernetes.io/default-container":      name,
	}
	if withMetricsExporter(cfg) {
		for k, v := range annotationsForSmbMetricsPod() {
			annotations[k] = v
		}
	}
	return annotations
}

func withMetricsExporter(cfg *conf.OperatorConfig) bool {
	return strings.ToLower(cfg.MetricsExporterMode) == "enabled"
}
