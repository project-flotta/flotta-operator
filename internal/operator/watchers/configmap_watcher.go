package watchers

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	clientwatch "k8s.io/client-go/tools/watch"
)

var (
	logger logr.Logger
)

// watcher implements config.Watch interface which will create a ConfigMapWatcher.
type watcher struct {
	configMapGetter v1.ConfigMapInterface
	configMapName   string
}

//go:generate mockgen -package=watchers -destination=mock_configmap.go k8s.io/client-go/kubernetes/typed/core/v1 ConfigMapInterface
//go:generate mockgen -package=watchers -destination=mock_watcher.go k8s.io/apimachinery/pkg/watch Interface
func WatchForChanges(cmi v1.ConfigMapInterface, configMapName string, dataField string, dataValue string, setupLogger logr.Logger, lastResourceVersion string, exitFunc func()) {
	logger = setupLogger
	logger.V(1).Info("watch for changes", "configMap name", configMapName, "data field", dataField, "current value", dataValue)

	wc := watcher{
		configMapGetter: cmi,
		configMapName:   configMapName,
	}

	w, err := clientwatch.NewRetryWatcher(lastResourceVersion, wc)
	if err != nil {
		logger.Error(err, "cannot create watcher", "configMap name", configMapName)
		exitFunc()
	}

	checkConfigMapChanges(w.ResultChan(), dataField, dataValue, exitFunc)
}

func checkConfigMapChanges(eventChannel <-chan watch.Event, dataField string, dataValue string, exitFunc func()) {
	for {
		event, open := <-eventChannel
		if open {
			switch event.Type {
			case watch.Added:
				fallthrough
			case watch.Modified:
				if updatedMap, ok := event.Object.(*corev1.ConfigMap); ok {
					if updatedValue, ok := updatedMap.Data[dataField]; ok {
						if updatedValue != dataValue {
							logger.Info("restarting pod to update the logging level", "current level", dataValue, "new level", updatedValue)
							exitFunc()
						}
					}
				}
			case watch.Deleted:
				fallthrough
			default:
				// Do nothing
			}
		} else {
			// If eventChannel is closed, it means the server has closed the connection
			return
		}
	}
}

func (w watcher) Watch(options metav1.ListOptions) (watch.Interface, error) {
	options.FieldSelector = fields.OneTermEqualSelector("metadata.name", w.configMapName).String()
	watcher, err := w.configMapGetter.Watch(context.Background(), options)

	return watcher, err

}
