package server

import (
	"crypto/tls"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/json"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	"net/http"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

var log = logf.Log.WithName("server")

type AdmissionController interface {
	HandleAdmission(review *v1beta1.AdmissionReview) error
}

type AdmissionControllerServer struct {
	AdmissionController AdmissionController
	Decoder             runtime.Decoder
}

func (a *AdmissionControllerServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if data, err := ioutil.ReadAll(r.Body); err == nil {
		body = data
	}
	review := &v1beta1.AdmissionReview{}
	_, _, err := a.Decoder.Decode(body, nil, review)
	if err != nil {
		log.Error(err, "Can't decode request")
	}

	_ = a.AdmissionController.HandleAdmission(review)

	responseInBytes, err := json.Marshal(review)
	if err != nil {
		log.Error(err, "Could not encode response")
	}

	if _, err := w.Write(responseInBytes); err != nil {
		log.Error(err, "")
	}
}

func GetAdmissionServerNoSSL(ac AdmissionController, listenOn int) *http.Server {

	admissionControllerHandler := AdmissionControllerServer{
		AdmissionController: ac,
		Decoder:             codecs.UniversalDeserializer(),
	}

	router := mux.NewRouter()
	router.PathPrefix("/mutate").Handler(&admissionControllerHandler)
	server := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%v", listenOn),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}

	return server
}

func GetAdmissionValidationServer(ac AdmissionController, tlsCert, tlsKey string, listenOn int) *http.Server {
	sCert, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
	if err != nil {
		log.Error(err, "Failed getting TLS cert/key")
	}

	server := GetAdmissionServerNoSSL(ac, listenOn)
	server.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{sCert},
	}

	return server
}
