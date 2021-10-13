package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HasFinalizer checks whether specific finalizer is set on a CR
func HasFinalizer(cr *metav1.ObjectMeta, name string) bool {
	for _, f := range cr.GetFinalizers() {
		if f == name {
			return true
		}
	}
	return false
}
