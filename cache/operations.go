package cache

import (
	"fmt"
	"math"

	aero "github.com/aerospike/aerospike-client-go"
	"github.com/sirupsen/logrus"
)

const (
	aerospikeNameSpace = "dev_bbcache"
	aerospikeSet       = "k8_annotation"
)

func AddRole(podIp, role, namespace string) error {
	key, gErr := getKey(podIp)
	if gErr != nil {
		rErr := fmt.Errorf("failed to prepare the aerospike key %v %v", podIp, gErr)
		logrus.Errorf(rErr.Error())

		return rErr
	}

	logrus.Infof("adding the bins with role: %v namespace: %v", role, namespace)
	bins := aero.BinMap{
		"role":      role,
		"namespace": namespace,
	}

	basePol := aero.BasePolicy{SendKey: true}
	pol := aero.WritePolicy{Expiration: math.MaxUint32}
	pol.BasePolicy = basePol

	client, cErr := getClient()
	if cErr != nil {
		rErr := fmt.Errorf("failed to connect to the aerospike key %v %v", podIp, cErr)
		logrus.Errorf(rErr.Error())

		return rErr
	}

	pErr := client.Put(&pol, key, bins)
	if pErr != nil {
		logrus.Errorf("failed to put the key %v %v", podIp, pErr.Error())

		return pErr
	}

	logrus.Infof("aerospike key set for key: %v", podIp)

	return nil
}

func UpdateRole(podIp, role, namespace string) error {
	key, gErr := getKey(podIp)
	if gErr != nil {
		rErr := fmt.Errorf("failed to prepare the aerospike key %v %v", podIp, gErr)
		logrus.Errorf(rErr.Error())

		return rErr
	}

	logrus.Infof("adding the bins with role: %v namespace: %v", role, namespace)
	bins := aero.BinMap{
		"role":      role,
		"namespace": namespace,
	}

	basePol := aero.BasePolicy{SendKey: true}
	pol := aero.WritePolicy{Expiration: math.MaxUint32}
	pol.BasePolicy = basePol

	client, cErr := getClient()
	if cErr != nil {
		rErr := fmt.Errorf("failed to connect to the aerospike key %v %v", podIp, cErr)
		logrus.Errorf(rErr.Error())

		return rErr
	}

	pErr := client.Put(&pol, key, bins)
	if pErr != nil {
		logrus.Errorf("failed to put the key %v %v", podIp, pErr.Error())

		return pErr
	}

	logrus.Infof("aerospike key set for key: %v", podIp)

	return nil
}

func DeleteRole(podIp string) error {
	key, gErr := getKey(podIp)
	if gErr != nil {
		rErr := fmt.Errorf("failed to prepare the aerospike key %v %v", podIp, gErr)
		logrus.Errorf(rErr.Error())

		return rErr
	}

	client, cErr := getClient()
	if cErr != nil {
		rErr := fmt.Errorf("failed to connect to the aerospike key %v %v", podIp, cErr)
		logrus.Errorf(rErr.Error())

		return rErr
	}

	keyDeleted, dErr := client.Delete(nil, key)
	if dErr != nil {
		logrus.Errorf("failed to put the key %v %v", podIp, dErr.Error())

		return dErr
	}

	logrus.Debugf("key deleted for key %v %v", podIp, keyDeleted)

	return nil
}

func GetRole(podIp string) (string, string, error) {
	key, gErr := getKey(podIp)
	if gErr != nil {
		rErr := fmt.Errorf("failed to prepare the aerospike key %v %v", podIp, gErr)
		logrus.Errorf(rErr.Error())

		return "", "", rErr
	}

	pol := aero.BasePolicy{SendKey: true}
	client, cErr := getClient()
	if cErr != nil {
		rErr := fmt.Errorf("failed to connect to the aerospike key %v %v", podIp, cErr)
		logrus.Errorf(rErr.Error())

		return "", "", cErr
	}

	record, gErr := client.Get(&pol, key)
	if gErr != nil {
		logrus.Errorf("failed to get the key %v %v", podIp, gErr.Error())

		return "", "", gErr
	}

	logrus.Infof("found value %v for key %v", record.Bins, podIp)

	role := record.Bins["role"].(string)
	namespace := record.Bins["namespace"].(string)

	logrus.Infof("found role: %v namespace: %v for key: %v", role, namespace, podIp)

	return role, namespace, nil
}

func getKey(podIp string) (*aero.Key, error) {
	logrus.Infof("preparing the aerospike key as %v", podIp)
	key, kErr := aero.NewKey(aerospikeNameSpace, aerospikeSet, podIp)
	if kErr != nil {
		logrus.Errorf("failed to get the aerospike key %v %v", podIp, kErr.Error())

		return nil, kErr
	}

	return key, nil
}
