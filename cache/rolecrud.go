package cache

import (
	aero "github.com/aerospike/aerospike-client-go"
	"github.com/sirupsen/logrus"
)

func AddRole(podIp, role, namespace string) error {
	bins := aero.BinMap{
		"role":      role,
		"namespace": namespace,
	}

	return Add(podIp, bins)
}

func UpdateRole(podIp, role, namespace string) error {
	return AddRole(podIp, role, namespace)
}

func DeleteRole(podIp string) error {

	return Delete(podIp)
}

func GetRole(podIp string) (string, string, error) {
	record, gErr := Get(podIp)
	if gErr != nil {
		logrus.Errorf("failed to get the key %v %v", podIp, gErr.Error())

		return "", "", gErr
	}

	logrus.Debugf("found value %v for key %v", record.Bins, podIp)

	role := record.Bins["role"].(string)
	namespace := record.Bins["namespace"].(string)

	logrus.Debugf("found role: %v namespace: %v for key: %v", role, namespace, podIp)

	return role, namespace, nil
}
