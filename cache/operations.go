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

func Add(key string, bins aero.BinMap) error {
	basePol := aero.BasePolicy{SendKey: true}
	pol := aero.WritePolicy{Expiration: math.MaxUint32}
	pol.BasePolicy = basePol

	client, cErr := getClient()
	if cErr != nil {
		rErr := fmt.Errorf("failed to connect to the aerospike key %v %v", key, cErr)
		logrus.Errorf(rErr.Error())

		return rErr
	}

	aeroKey, gErr := prepareKey(key)
	if gErr != nil {
		rErr := fmt.Errorf("failed to prepare the aerospike key %v %v", key, gErr)
		logrus.Errorf(rErr.Error())

		return gErr
	}

	pErr := client.Put(&pol, aeroKey, bins)
	if pErr != nil {
		logrus.Errorf("failed to put the key %v %v", aeroKey.Value(), pErr.Error())

		return pErr
	}

	logrus.Debugf("aerospike key set for key: %v", aeroKey.Value())

	return nil
}

func Update(key string, bins aero.BinMap) error {
	return Add(key, bins)
}

func Delete(key string) error {
	client, cErr := getClient()
	if cErr != nil {
		rErr := fmt.Errorf("failed to connect to the aerospike key %v %v", key, cErr)
		logrus.Errorf(rErr.Error())

		return rErr
	}

	aeroKey, gErr := prepareKey(key)
	if gErr != nil {
		rErr := fmt.Errorf("failed to prepare the aerospike key %v %v", key, gErr)
		logrus.Errorf(rErr.Error())

		return gErr
	}

	keyDeleted, dErr := client.Delete(nil, aeroKey)
	if dErr != nil {
		logrus.Errorf("failed to put the key %v %v", aeroKey.Value(), dErr.Error())

		return dErr
	}

	logrus.Debugf("key deleted for key %v %v", aeroKey.Value(), keyDeleted)

	return nil
}

func Get(key string) (*aero.Record, error) {
	pol := aero.BasePolicy{SendKey: true}
	client, cErr := getClient()
	if cErr != nil {
		rErr := fmt.Errorf("failed to connect to the aerospike key %v %v", key, cErr)
		logrus.Errorf(rErr.Error())

		return nil, cErr
	}

	aeroKey, gErr := prepareKey(key)
	if gErr != nil {
		rErr := fmt.Errorf("failed to prepare the aerospike key %v %v", key, gErr)
		logrus.Errorf(rErr.Error())

		return nil, gErr
	}

	record, getErr := client.Get(&pol, aeroKey)
	if getErr != nil {
		logrus.Errorf("failed to get the key %v %v", aeroKey.Value(), gErr.Error())

		return nil, gErr
	}

	logrus.Debugf("found value %v for key %v", record.Bins, aeroKey.Value())

	return record, nil
}

func prepareKey(userKey string) (*aero.Key, error) {
	key, kErr := aero.NewKey(aerospikeNameSpace, aerospikeSet, userKey)
	if kErr != nil {
		logrus.Errorf("failed to prepare the aerospike key %v %v", userKey, kErr.Error())

		return nil, kErr
	}

	return key, nil
}
