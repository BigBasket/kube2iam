package server

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jtblin/kube2iam"
	"github.com/jtblin/kube2iam/iam"
	"github.com/jtblin/kube2iam/k8s"
	"github.com/jtblin/kube2iam/mappings"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"
)

const (
	defaultAppPort                    = "8181"
	defaultCacheSyncAttempts          = 10
	defaultIAMRoleKey                 = "iam.amazonaws.com/role"
	defaultIAMExternalID              = "iam.amazonaws.com/external-id"
	defaultLogLevel                   = "info"
	defaultLogFormat                  = "text"
	defaultMaxElapsedTime             = 1 * time.Second
	defaultIAMRoleSessionTTL          = 15 * time.Minute
	defaultMaxInterval                = 1 * time.Second
	defaultMetadataAddress            = "169.254.169.254"
	defaultNamespaceKey               = "iam.amazonaws.com/allowed-roles"
	defaultCacheResyncPeriod          = 30 * time.Minute
	defaultResolveDupIPs              = false
	defaultNamespaceRestrictionFormat = "glob"
	healthcheckInterval               = 30 * time.Second
	defaultStsVpcEndpoint             = ""
)

var tokenRouteRegexp = regexp.MustCompile("^/?[^/]+/api/token$")

// Server encapsulates all of the parameters necessary for starting up
// the server. These can either be set via command line or directly.
type Server struct {
	APIServer                  string
	APIToken                   string
	AppPort                    string
	MetricsPort                string
	BaseRoleARN                string
	DefaultIAMRole             string
	IAMRoleKey                 string
	IAMExternalID              string
	IAMRoleSessionTTL          time.Duration
	MetadataAddress            string
	HostInterface              string
	HostIP                     string
	NodeName                   string
	NamespaceKey               string
	CacheResyncPeriod          time.Duration
	LogLevel                   string
	LogFormat                  string
	NamespaceRestrictionFormat string
	ResolveDupIPs              bool
	UseRegionalStsEndpoint     bool
	AddIPTablesRule            bool
	AutoDiscoverBaseArn        bool
	AutoDiscoverDefaultRole    bool
	Debug                      bool
	Insecure                   bool
	NamespaceRestriction       bool
	Verbose                    bool
	Version                    bool
	iam                        *iam.Client
	k8s                        *k8s.Client
	roleMapper                 *mappings.RoleMapper
	BackoffMaxElapsedTime      time.Duration
	BackoffMaxInterval         time.Duration
	InstanceID                 string
	HealthcheckFailReason      string
	StsVpcEndPoint             string
	BootAsWebServer            bool
	BootAsWatcher              bool
}

func parseRemoteAddr(addr string) string {
	n := strings.IndexByte(addr, ':')
	if n <= 1 {
		return ""
	}
	hostname := addr[0:n]
	if net.ParseIP(hostname) == nil {
		return ""
	}
	return hostname
}

func (s *Server) getRoleMapping(IP string) (*mappings.RoleMappingResult, error) {
	var roleMapping *mappings.RoleMappingResult
	var err error

	roleMapping, err = s.roleMapper.GetRoleMappingUsingCache(IP)

	if err != nil {
		return nil, err
	}

	return roleMapping, nil
}

// Run runs the specified Server.
func (s *Server) Run(host, token, nodeName string, insecure bool) error {
	k, err := k8s.NewClient(host, token, nodeName, insecure, s.ResolveDupIPs)
	if err != nil {
		return err
	}

	s.k8s = k
	var nErr error
	s.iam, nErr = iam.NewClient(s.BaseRoleARN, s.UseRegionalStsEndpoint, s.StsVpcEndPoint)
	if nErr != nil {
		return nErr
	}

	s.roleMapper = mappings.NewRoleMapper(s.IAMRoleKey, s.IAMExternalID, s.DefaultIAMRole, s.NamespaceRestriction,
		s.NamespaceKey, s.iam, s.k8s, s.NamespaceRestrictionFormat)

	if s.BootAsWatcher {
		wg := new(sync.WaitGroup)
		wg.Add(1)

		go func() {
			log.Debugf("Starting pod and namespace sync jobs with %s resync period", s.CacheResyncPeriod.String())
			podSynched := s.k8s.WatchForPods(
				kube2iam.NewPodHandler(s.IAMRoleKey, s.DefaultIAMRole, s.NamespaceKey, s.iam), s.CacheResyncPeriod)
			namespaceSynched := s.k8s.WatchForNamespaces(kube2iam.NewNamespaceHandler(s.NamespaceKey), s.CacheResyncPeriod)

			synced := false
			for i := 0; i < defaultCacheSyncAttempts && !synced; i++ {
				synced = cache.WaitForCacheSync(nil, podSynched, namespaceSynched)
			}

			if !synced {
				log.Fatalf("Attempted to wait for caches to be synced for %d however it is not done.  Giving up.", defaultCacheSyncAttempts)
			} else {
				log.Debugln("Caches have been synced.  Proceeding with server.")
			}
		}()

		wg.Wait()
	} else if s.BootAsWebServer {
		startServer(s)
	}

	return nil
}

// NewServer will create a new Server with default values.
func NewServer() *Server {
	return &Server{
		AppPort:                    defaultAppPort,
		MetricsPort:                defaultAppPort,
		BackoffMaxElapsedTime:      defaultMaxElapsedTime,
		IAMRoleKey:                 defaultIAMRoleKey,
		IAMExternalID:              defaultIAMExternalID,
		BackoffMaxInterval:         defaultMaxInterval,
		LogLevel:                   defaultLogLevel,
		LogFormat:                  defaultLogFormat,
		MetadataAddress:            defaultMetadataAddress,
		NamespaceKey:               defaultNamespaceKey,
		CacheResyncPeriod:          defaultCacheResyncPeriod,
		ResolveDupIPs:              defaultResolveDupIPs,
		NamespaceRestrictionFormat: defaultNamespaceRestrictionFormat,
		HealthcheckFailReason:      "",
		IAMRoleSessionTTL:          defaultIAMRoleSessionTTL,
		StsVpcEndPoint:             defaultStsVpcEndpoint,
		BootAsWebServer:            false,
		BootAsWatcher:              false,
	}
}

func (s *Server) getInstanceRole(context echo.Context) error {
	context.Response().Header().Set("Server", "EC2ws")

	log.Infof("processing request for instance role ip: %v", context.Request().RemoteAddr)
	roleMapping, err := s.getRoleMapping(parseRemoteAddr(context.Request().RemoteAddr))
	if err != nil {
		context.String(http.StatusInternalServerError, err.Error())

		return err
	}

	// If a base ARN has been supplied and this is not cross-account then
	// return a simple role-name, otherwise return the full ARN
	if s.iam.BaseARN != "" && strings.HasPrefix(roleMapping.Role, s.iam.BaseARN) {
		context.String(http.StatusOK, strings.TrimPrefix(roleMapping.Role, s.iam.BaseARN))

		return nil
	}

	context.String(http.StatusOK, roleMapping.Role)

	return nil
}

func (s *Server) getPodRole(context echo.Context) error {
	context.Response().Header().Set("Server", "EC2ws")

	wantedRole := context.Param("role")
	wantedRoleARN := s.iam.RoleARN(wantedRole)

	roleLogger := log.WithFields(log.Fields{
		"pod.iam.role": wantedRole,
	})

	credentials, err := s.iam.AssumeRole(wantedRoleARN, "", parseRemoteAddr(context.Request().RemoteAddr), s.IAMRoleSessionTTL)
	if err != nil {
		roleLogger.Errorf("Error assuming role %+v", err)
		context.String(http.StatusServiceUnavailable, err.Error())

		return err
	}

	roleLogger.Debugf("retrieved credentials from sts endpoint: %s", s.iam.Endpoint)

	result, _ := json.Marshal(credentials)

	return context.JSONBlob(http.StatusOK, result)
}

func (s *Server) allAWSOtherRoutes(context echo.Context) error {
	// Remove remoteaddr to prevent issues with new IMDSv2 to fail when x-forwarded-for header is present
	// for more details please see: https://github.com/aws/aws-sdk-ruby/issues/2177 https://github.com/uswitch/kiam/issues/359
	token := context.Request().Header.Get("X-aws-ec2-metadata-token")
	if (context.Request().Method == http.MethodPut &&
		tokenRouteRegexp.MatchString(context.Request().URL.Path)) || (context.Request().Method == http.MethodGet && token != "") {
		context.Request().RemoteAddr = ""
	}

	proxy := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: s.MetadataAddress})
	proxy.ServeHTTP(context.Response().Writer, context.Request())

	log.WithField("metadata.url", s.MetadataAddress).Debug("Proxy ec2 metadata request")

	return nil
}
