package server

import (
	"context"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/vmware/generic-sidecar-injector/pkg/apis/vmware/v1alpha1"
	"io"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	"net/http/httptest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"
)

var (
	AdmissionRequestNS = v1beta1.AdmissionReview{
		TypeMeta: v1.TypeMeta{
			Kind: "AdmissionReview",
		},
		Request: &v1beta1.AdmissionRequest{
			UID: "e911857d-c318-11e8-bbad-025000000001",
			Kind: v1.GroupVersionKind{
				Kind: "Namespace",
			},
			Operation: "CREATE",
			Object: runtime.RawExtension{
				Raw: []byte(`{"metadata": {
        						"name": "test",
        						"uid": "e911857d-c318-11e8-bbad-025000000001",
						        "creationTimestamp": "2018-09-28T12:20:39Z"
      						}}`),
			},
		},
	}
)

func decodeResponse(body io.ReadCloser) *v1beta1.AdmissionReview {
	response, _ := ioutil.ReadAll(body)
	review := &v1beta1.AdmissionReview{}
	_, _, _ = codecs.UniversalDeserializer().Decode(response, nil, review)
	return review
}

func encodeRequest(review *v1beta1.AdmissionReview) []byte {
	ret, err := json.Marshal(review)
	if err != nil {
		logrus.Errorln(err)
	}
	return ret
}

func TestServeReturnsCorrectJson(t *testing.T) {
	sidecar := &v1alpha1.Sidecar{
		ObjectMeta: v1.ObjectMeta{
			Name:      "sidecar-to-inject",
			Namespace: "test",
		},
		Spec: v1alpha1.SidecarSpec{
			Containers: []corev1.Container{
				{
					Name:            "test-container",
					Image:           "telegraf:0.0.1",
					ImagePullPolicy: "Always",
				},
			},
		},
	}

	// Objects to track in the fake client.
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.Sidecar{})
	objs := []runtime.Object{sidecar}
	cl := fake.NewFakeClientWithScheme(scheme, objs...)
	instance := &v1alpha1.SidecarList{}
	_ = cl.List(context.TODO(), instance)

	mutateAdmissionController := &MutateAdmission{Client: cl}
	server := httptest.NewServer(GetAdmissionServerNoSSL(mutateAdmissionController, 8080).Handler)
	requestString := string(encodeRequest(&AdmissionRequestNS))
	myr := strings.NewReader(requestString)
	r, _ := http.Post(server.URL, "application/json", myr)
	review := decodeResponse(r.Body)

	if review.Request.UID != AdmissionRequestNS.Request.UID {
		t.Error("Request and response UID don't match")
	}
}
