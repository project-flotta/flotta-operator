package watchers

import (
	"context"
	"os"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

var (
	logger  logr.Logger
	backoff = wait.Backoff{
		Steps:    10,
		Duration: 10 * time.Second,
		Factor:   2.0,
		Jitter:   0.1,
		Cap:      2 * time.Hour,
	}
)

func WatchForChanges(clientset kubernetes.Interface, namespace string, configMapName string, dataField string, dataValue string, setupLogger logr.Logger) {
	logger = setupLogger
	logger.V(1).Info("watch for changes", "namespace", namespace, "configMap name", configMapName, "data field", dataField, "current value", dataValue)
	for {
		var watcher watch.Interface
		err := createWatcher(clientset, namespace, configMapName, &watcher)
		if err != nil {
			logger.Error(err, "cannot create watcher", "namespace", namespace, "configMap name", configMapName)
			os.Exit(1)
		}

		checkConfigMapChanges(watcher.ResultChan(), dataField, dataValue)
	}
}

func checkConfigMapChanges(eventChannel <-chan watch.Event, dataField string, dataValue string) {
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
							os.Exit(1)
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

func createWatcher(clientset kubernetes.Interface, namespace string, configMapName string, watcher *watch.Interface) error {
	err := retry.OnError(backoff, retriable, func() error {
		var innerErr error
		*watcher, innerErr = clientset.CoreV1().ConfigMaps(namespace).Watch(context.TODO(),
			metav1.SingleObject(metav1.ObjectMeta{Name: configMapName, Namespace: namespace}))
		if innerErr != nil {
			logger.Info("cannot create watcher", "namespace", namespace, "configMap name", configMapName, "error", innerErr)
		}
		return innerErr
	})
	return err
}

func retriable(err error) bool {
	retry := err != nil
	logger.V(1).Info("cannot create watcher", "retriable", retry, "error", err)
	return retry
}
