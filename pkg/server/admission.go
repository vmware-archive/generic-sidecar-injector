package server

import (
	"context"
	"encoding/json"
	"github.com/vmware/generic-sidecar-injector/pkg/apis/vmware/v1alpha1"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ignoredNamespaces = map[string]bool{
	v1.NamespaceSystem: true,
	v1.NamespacePublic: true,
}

var injectOptions = map[string]bool{
	"y":       true,
	"yes":     true,
	"true,":   true,
	"on":      true,
	"enabled": true,
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type MutateAdmission struct {
	Client client.Client
}

func isMutationRequired(metadata *v1.ObjectMeta, sidecars []v1alpha1.Sidecar) (bool, v1alpha1.Sidecar) {

	if ignoredNamespaces[metadata.Namespace] {
		log.Info("Skipping %v in namespace %v", metadata.Name, metadata.Namespace)
		return false, v1alpha1.Sidecar{}
	}

	annotations := metadata.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	for _, sidecar := range sidecars {
		inject := annotations[sidecar.Name+"/inject"]
		if injectOptions[inject] {
			return true, sidecar
		}
	}

	return false, v1alpha1.Sidecar{}
}

func addContainer(target, added []corev1.Container, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Container{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func addVolume(target, added []corev1.Volume, basePath string) (patch []patchOperation) {
	first := len(target) == 0
	var value interface{}
	for _, add := range added {
		value = add
		path := basePath
		if first {
			first = false
			value = []corev1.Volume{add}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func updateAnnotation(target map[string]string, added map[string]string) (patch []patchOperation) {
	for key, value := range added {
		if target[key] != value {
			target[key] = value
		}
	}
	patch = append(patch, patchOperation{
		Op:    "add",
		Path:  "/metadata/annotations",
		Value: target,
	})
	return patch
}

func createPatch(pod *corev1.Pod, sidecar v1alpha1.Sidecar, annotations map[string]string) ([]byte, error) {
	var patch []patchOperation

	patch = append(patch, addContainer(pod.Spec.Containers, sidecar.Spec.Containers, "/spec/containers")...)
	patch = append(patch, addVolume(pod.Spec.Volumes, sidecar.Spec.Volumes, "/spec/volumes")...)
	patch = append(patch, updateAnnotation(pod.Annotations, annotations)...)

	return json.Marshal(patch)
}

func (m *MutateAdmission) HandleAdmission(review *v1beta1.AdmissionReview) error {
	req := review.Request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		log.Error(err, "Could not unmarshal raw object")
		review.Response = &v1beta1.AdmissionResponse{
			Result: &v1.Status{
				Message: err.Error(),
			},
		}
		return nil
	}

	// Get list of current Sidecars
	sidecars := &v1alpha1.SidecarList{}
	_ = m.Client.List(context.Background(), sidecars)

	mutationRequired, sidecar := isMutationRequired(&pod.ObjectMeta, sidecars.Items)
	if !mutationRequired {
		review.Response = &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	annotations := map[string]string{sidecar.Name + "/status": "injected"}
	patchBytes, err := createPatch(&pod, sidecar, annotations)
	if err != nil {
		review.Response = &v1beta1.AdmissionResponse{
			Result: &v1.Status{
				Message: err.Error(),
			},
		}
	}

	review.Response = &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}

	return nil
}
