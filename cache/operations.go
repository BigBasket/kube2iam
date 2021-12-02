package cache

import (
	aero "github.com/aerospike/aerospike-client-go"
	"github.com/sirupsen/logrus"
)

const (
	nameSpace = "dev_bbcache"
	set       = "pod_annotation_role"
)

func AddRole(podIp, role, namespace string) error {
	key := getKey(podIp)

	bins := aero.BinMap{
		"role":      role,
		"namespace": namespace,
	}

	writePolicy := aero.NewWritePolicy(0, 0)
	pErr := getClient().Put(writePolicy, key, bins)
	if pErr != nil {
		logrus.Errorf("failed to put the key %v %v", podIp, pErr.Error())
	}

	return pErr
}

func UpdateRole(podIp, role, namespace string) error {
	key := getKey(podIp)

	bins := aero.BinMap{
		"role":      role,
		"namespace": namespace,
	}

	writePolicy := aero.NewWritePolicy(0, 0)
	pErr := getClient().Put(writePolicy, key, bins)
	if pErr != nil {
		logrus.Errorf("failed to put the key %v %v", podIp, pErr.Error())
	}

	return pErr
}

func DeleteRole(podIp string) error {
	key := getKey(podIp)

	writePolicy := aero.NewWritePolicy(0, 0)
	keyDeleted, dErr := getClient().Delete(writePolicy, key)
	if dErr != nil {
		logrus.Errorf("failed to put the key %v %v", podIp, dErr.Error())
	}

	logrus.Debugf("key deleted for key %v %v", podIp, keyDeleted)

	return dErr
}

func GetRole(podIp string) (*string, *string, error) {
	key := getKey(podIp)
	readPolicy := aero.NewPolicy()
	record, gErr := getClient().Get(readPolicy, key)
	if gErr != nil {
		logrus.Errorf("failed to get the key %v %v", podIp, gErr.Error())

		return nil, nil, gErr
	}
	role := record.Bins["role"].(string)
	namespace := record.Bins["namespace"].(string)

	return &role, &namespace, nil
}

func getKey(podIp string) *aero.Key {
	key, _ := aero.NewKey(nameSpace, set, podIp)

	return key
}
