package utils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GetRef func(interface{}) (bool, *bool, string)
type GetRefEnv func(env corev1.EnvVar) (bool, *bool, string, string)

type StringSet = map[string]interface{}
type MapType = map[string]StringSet

// HasFinalizer checks whether specific finalizer is set on a CR
func HasFinalizer(cr *metav1.ObjectMeta, name string) bool {
	for _, f := range cr.GetFinalizers() {
		if f == name {
			return true
		}
	}
	return false
}

// NormalizeLabel normalizes label string according to k8s format
func NormalizeLabel(name string) (string, error) {
	// convert name to lowercase
	name = strings.ToLower(name)

	// slice string based on first and last alphanumeric character
	firstLegal := strings.IndexFunc(name, func(c rune) bool { return unicode.IsLower(c) || unicode.IsDigit(c) })
	lastLegal := strings.LastIndexFunc(name, func(c rune) bool { return unicode.IsLower(c) || unicode.IsDigit(c) })

	if firstLegal < 0 {
		return "", fmt.Errorf("The name doesn't contain a legal alphanumeric character")
	}

	name = name[firstLegal : lastLegal+1]
	reg := regexp.MustCompile("[^a-z0-9-_.]+")
	name = reg.ReplaceAllString(name, "")

	return name, nil
}

func ExtractInfoFromEnvFrom(envs []corev1.EnvFromSource, themap MapType, ref GetRef) {
	for _, env := range envs {
		exist, opt, name := ref(env)
		if !exist {
			continue
		}
		optional := false
		if opt != nil {
			optional = *opt
		}
		if keys, ok := themap[name]; ok {
			if !optional && keys == nil {
				themap[name] = StringSet{}
			}
		} else {
			if optional {
				themap[name] = nil
			} else {
				themap[name] = StringSet{}
			}
		}
	}
}

func ExtractInfoFromVolume(volumes []corev1.Volume, themap MapType, ref GetRef) {
	for _, volume := range volumes {
		exist, opt, name := ref(volume)
		if !exist {
			continue
		}
		optional := false
		if opt != nil {
			optional = *opt
		}
		if keys, ok := themap[name]; ok {
			if !optional && keys == nil {
				themap[name] = StringSet{}
			}
		} else {
			if optional {
				themap[name] = nil
			} else {
				themap[name] = StringSet{}
			}
		}
	}
}

func ExtractInfoFromEnv(env []corev1.EnvVar, configmapMap MapType, ref GetRefEnv) {
	for _, envVar := range env {
		exist, opt, name, key := ref(envVar)

		if envVar.ValueFrom == nil || !exist {
			continue
		}
		var optional bool
		if opt != nil {
			optional = *opt
		}
		if keys, ok := configmapMap[name]; ok {
			if !optional {
				if keys == nil {
					configmapMap[name] = StringSet{key: nil}
				} else {
					keys[key] = nil
				}
			}
		} else {
			if optional {
				configmapMap[name] = nil
			} else {
				configmapMap[name] = StringSet{key: nil}
			}
		}
	}
}

// FilterByPodmanPrefix does two things:
// first it filters the keys that start with prefix "podman/"
// And secondly, of those keys that have the "podman/" prefix, it uses the remaining part of the key as the new key in the returning map with the same value as in the original map
// e.g. 'podman/this-is-my-key:1' is stored in the returning map as 'this-is-my-key:1'
func FilterByPodmanPrefix(keyValues map[string]string) map[string]string {
	ret := make(map[string]string, len(keyValues))
	for key, value := range keyValues {
		if strings.HasPrefix(key, "podman/") {
			ret[key[7:]] = value
		}
	}
	return ret
}
