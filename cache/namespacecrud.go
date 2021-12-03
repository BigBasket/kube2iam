package cache

import (
	aero "github.com/aerospike/aerospike-client-go"
	"github.com/sirupsen/logrus"
)

func AddNRole(namespace string, role []string) error {
	bins := aero.BinMap{
		"role": role,
	}

	return Add(namespace, bins)
}

func UpdateNRole(namespace string, role []string) error {
	return AddNRole(namespace, role)
}

func DeleteNRole(namespace string) error {
	return Delete(namespace)
}

func GetNRole(namespace string) ([]string, error) {
	record, gErr := Get(namespace)
	if gErr != nil {
		logrus.Errorf("failed to get the key %v %v", namespace, gErr.Error())

		return []string{}, gErr
	}

	logrus.Debugf("found value %v for key %v", record.Bins, namespace)

	role := record.Bins["role"].([]string)

	logrus.Debugf("found role: %v namespace: %v", role, namespace)

	return role, nil
}
