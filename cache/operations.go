package cache

import (
	"math"

	"github.com/aerospike/aerospike-client-go"
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

	basePol := aerospike.BasePolicy{SendKey: true}
	pol := aerospike.WritePolicy{Expiration: math.MaxUint32}
	pol.BasePolicy = basePol

	pErr := getClient().Put(&pol, key, bins)
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

	basePol := aerospike.BasePolicy{SendKey: true}
	pol := aerospike.WritePolicy{Expiration: math.MaxUint32}
	pol.BasePolicy = basePol

	pErr := getClient().Put(&pol, key, bins)
	if pErr != nil {
		logrus.Errorf("failed to put the key %v %v", podIp, pErr.Error())
	}

	return pErr
}

func DeleteRole(podIp string) error {
	key := getKey(podIp)

	keyDeleted, dErr := getClient().Delete(nil, key)
	if dErr != nil {
		logrus.Errorf("failed to put the key %v %v", podIp, dErr.Error())
	}

	logrus.Debugf("key deleted for key %v %v", podIp, keyDeleted)

	return dErr
}

func GetRole(podIp string) (*string, *string, error) {
	key := getKey(podIp)
	pol := aerospike.BasePolicy{SendKey: true}
	record, gErr := getClient().Get(&pol, key)

	if gErr != nil {
		logrus.Errorf("failed to get the key %v %v", podIp, gErr.Error())

		return nil, nil, gErr
	}
	role := record.Bins["role"].(string)
	namespace := record.Bins["namespace"].(string)

	return &role, &namespace, nil
}

func getKey(podIp string) *aero.Key {
	key, kErr := aero.NewKey(nameSpace, set, podIp)
	logrus.Errorf("failed to get the aerospike key %v %v", podIp, kErr.Error())

	return key
}
