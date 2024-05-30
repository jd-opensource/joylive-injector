package admission

import (
	"fmt"
	"github.com/jd-opensource/joylive-injector/pkg/log"
	"io"
	"net/http"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/runtime/serializer"

	jsoniter "github.com/json-iterator/go"

	"github.com/jd-opensource/joylive-injector/pkg/route"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type AdmissionType string

const (
	AdmissionTypeMutating   AdmissionType = "Mutating"
	AdmissionTypeValidating AdmissionType = "Validating"
)

// AdmissionFunc defines an admission control handler
type AdmissionFunc struct {
	Type AdmissionType
	Path string
	Func func(request *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error)
}

// admissionFuncMap is a collection of global admission control handlers
type admissionFuncMap map[string]AdmissionFunc

var funcMap = make(admissionFuncMap, 10)

var admissionOnce sync.Once
var deserializer runtime.Decoder

// Setup initialize deserializer and register admission control handlers
// to the global routing handlers collection.
func Setup() {
	admissionOnce.Do(func() {
		log.Info("init kube deserializer...")
		deserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()

		log.Info("init admission func...")
		for p, af := range funcMap {
			log.Infof("load admission func: %s", af.Path)
			handlePath := strings.Replace(p, "_", "-", -1)
			if p != handlePath {
				log.Warnf("admission func handler path does not support '_', it has been automatically converted to '-'(%s => %s)", p, handlePath)
			}
			log.Infof("register admission path is: %s, p is %s ", handlePath, p)
			copyAf := af
			route.RegisterHandler(route.HandleFunc{
				Path:   handlePath,
				Method: http.MethodPost,
				Func: func(w http.ResponseWriter, r *http.Request) {
					defer func() { _ = r.Body.Close() }()
					log.Debugf("received post request: %s %s", r.Method, r.URL.Path)
					reqBs, err := io.ReadAll(r.Body)
					if err != nil {
						route.ResponseErr(handlePath, err.Error(), http.StatusInternalServerError, w)
						return
					}
					if reqBs == nil || len(reqBs) == 0 {
						route.ResponseErr(handlePath, "request body is empty", http.StatusBadRequest, w)
						return
					}
					log.Debugf("request body: %s", string(reqBs))

					reqReview := admissionv1.AdmissionReview{}
					if _, _, err := deserializer.Decode(reqBs, nil, &reqReview); err != nil {
						route.ResponseErr(handlePath, fmt.Sprintf("failed to decode req: %s", err), http.StatusInternalServerError, w)
						return
					}
					if reqReview.Request == nil {
						route.ResponseErr(handlePath, "admission review request is empty", http.StatusBadRequest, w)
						return
					}

					resp, err := copyAf.Func(reqReview.Request)
					if err != nil {
						route.ResponseErr(handlePath, fmt.Sprintf("admission func response: %s", err), http.StatusForbidden, w)
						return
					}
					if resp == nil {
						route.ResponseErr(handlePath, "admission func response is empty", http.StatusInternalServerError, w)
						return
					}
					resp.UID = reqReview.Request.UID
					respReview := admissionv1.AdmissionReview{
						TypeMeta: reqReview.TypeMeta,
						Response: resp,
					}
					respBs, err := jsoniter.Marshal(respReview)
					if err != nil {
						route.ResponseErr(handlePath, fmt.Sprintf("failed to marshal response: %s", err), http.StatusInternalServerError, w)
						log.Errorf("the expected response is: %v", respReview)
						return
					}

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, err = w.Write(respBs)
					log.Debugf("write response: %d: %s: %v", http.StatusOK, string(respBs), err)
				},
			})

		}

	})
}

func Register(af AdmissionFunc) {
	log.Infof("start to register admission func: %s", af.Path)
	if af.Path == "" {
		log.Fatalf("admission func path is empty")
	}

	if af.Type == "" {
		log.Fatalf("admission func type is empty")
	}

	handlePath := strings.ToLower(af.Path)
	if !strings.HasPrefix(handlePath, "/") {
		handlePath = "/" + handlePath
	}
	switch af.Type {
	case AdmissionTypeMutating:
		handlePath = "/mutating" + handlePath
	case AdmissionTypeValidating:
		handlePath = "/validating" + handlePath
	default:
		log.Fatalf("unsupported admission func type")
	}
	registeredAf, exist := funcMap[handlePath]
	if exist && registeredAf.Type == af.Type {
		log.Fatalf("admission func [%s], type: %s already registered", af.Path, af.Type)
	}
	funcMap[handlePath] = af
}
