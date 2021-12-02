package cache

import (
	"fmt"

	aero "github.com/aerospike/aerospike-client-go"
	"github.com/sirupsen/logrus"
)

const (
	host = "10.2.1.224"
)

var aeroClient *aero.Client = nil

func getClient() (*aero.Client, error) {
	if aeroClient == nil {
		var cErr error
		aeroClient, cErr = aero.NewClient(host, 3000)
		if cErr != nil {
			rErr := fmt.Errorf("failed to open the aerospike connection %v", cErr.Error())
			logrus.Errorf(rErr.Error())

			return nil, rErr
		}
	}

	return aeroClient, nil
}
