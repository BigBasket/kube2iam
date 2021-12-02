package kube2iam

import (
	"fmt"

	aerospike "github.com/jtblin/kube2iam/cache"
	"github.com/jtblin/kube2iam/iam"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

// PodHandler represents a pod handler.
type PodHandler struct {
	iamRoleKey     string
	defaultRoleARN string
	client         *iam.Client
}

func (p *PodHandler) podFields(pod *v1.Pod) log.Fields {
	return log.Fields{
		"pod.name":         pod.GetName(),
		"pod.namespace":    pod.GetNamespace(),
		"pod.status.ip":    pod.Status.PodIP,
		"pod.status.phase": pod.Status.Phase,
		"pod.iam.role":     pod.GetAnnotations()[p.iamRoleKey],
	}
}

// OnAdd is called when a pod is added.
func (p *PodHandler) OnAdd(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		log.Errorf("Expected Pod but OnAdd handler received %+v", obj)
		return
	}
	logger := log.WithFields(p.podFields(pod))

	podRole, gErr := p.getPodRole(pod)
	if gErr != nil {
		logger.Errorf("failed to get the pod role %v", gErr.Error())
	}

	aerospike.AddRole(pod.Status.PodIP, pod.GetNamespace(), podRole)

	//TODO JRN: Should we be filtering this by the `isPodActive` to reduce chatter and confusion about
	// what is actually being indexed by the indexer? This gets a little tricky with the OnUpdate piece
	// of cronjobs that stick around in Completed/Succeeded status
	logger.Debug("Pod OnAdd")
}

// OnUpdate is called when a pod is modified.
func (p *PodHandler) OnUpdate(oldObj, newObj interface{}) {
	_, ok1 := oldObj.(*v1.Pod)
	newPod, ok2 := newObj.(*v1.Pod)
	if !ok1 || !ok2 {
		log.Errorf("Expected Pod but OnUpdate handler received %+v %+v", oldObj, newObj)
		return
	}

	logger := log.WithFields(p.podFields(newPod))
	podRole, gErr := p.getPodRole(newPod)
	if gErr != nil {
		logger.Errorf("failed to get the pod role %v", gErr.Error())
	}
	aerospike.UpdateRole(newPod.Status.PodIP, newPod.GetNamespace(), podRole)

	logger.Debug("Pod OnUpdate")
}

// OnDelete is called when a pod is deleted.
func (p *PodHandler) OnDelete(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		deletedObj, dok := obj.(cache.DeletedFinalStateUnknown)
		if dok {
			pod, ok = deletedObj.Obj.(*v1.Pod)
		}
	}

	if !ok {
		log.Errorf("Expected Pod but OnDelete handler received %+v", obj)
		return
	}

	logger := log.WithFields(p.podFields(pod))
	aerospike.DeleteRole(pod.Status.PodIP)

	logger.Debug("Pod OnDelete")
}

func isPodActive(p *v1.Pod) bool {
	return p.Status.PodIP != "" &&
		v1.PodSucceeded != p.Status.Phase &&
		v1.PodFailed != p.Status.Phase
}

// PodIPIndexFunc maps a given Pod to it's IP for caching.
func PodIPIndexFunc(obj interface{}) ([]string, error) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return nil, fmt.Errorf("obj not pod: %+v", obj)
	}
	if isPodActive(pod) {
		return []string{pod.Status.PodIP}, nil
	}
	return nil, nil
}

// NewPodHandler constructs a pod handler given the relevant IAM Role Key
func NewPodHandler(iamRoleKey string, defaultRole string, client *iam.Client) *PodHandler {
	return &PodHandler{
		iamRoleKey:     iamRoleKey,
		defaultRoleARN: defaultRole,
		client:         client,
	}
}

func (p *PodHandler) getPodRole(pod *v1.Pod) (string, error) {
	rawRoleName, annotationPresent := pod.GetAnnotations()[p.iamRoleKey]

	if !annotationPresent && p.defaultRoleARN == "" {
		return "", fmt.Errorf("unable to find role for IP %s", pod.Status.PodIP)
	}

	if !annotationPresent {
		log.Warnf("Using fallback role for IP %s", pod.Status.PodIP)
		rawRoleName = p.defaultRoleARN
	}

	return p.client.RoleARN(rawRoleName), nil
}
