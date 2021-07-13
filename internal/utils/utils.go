package utils

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HasFinalizer checks whether specific finalizer is set on VM import CR
func HasFinalizer(cr *metav1.ObjectMeta, name string) bool {
	for _, f := range cr.GetFinalizers() {
		if f == name {
			return true
		}
	}
	return false
}

func Copy(from interface{}, to interface{}) error {
	if from == nil {
		return nil
	}
	jsonFrom, err := json.Marshal(from)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonFrom, to)
}
