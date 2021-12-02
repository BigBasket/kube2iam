package cache

import (
	aero "github.com/aerospike/aerospike-client-go"
	"github.com/sirupsen/logrus"
)

const (
	host = "10.2.1.224"
)

var aeroClient *aero.Client = nil

func getClient() *aero.Client {
	if aeroClient == nil {
		var cErr error
		aeroClient, cErr = aero.NewClient(host, 3000)
		if cErr != nil {
			logrus.Errorf("failed to open the aerospike connection %v", cErr.Error())
		}

		return nil
	}

	return aeroClient
}
