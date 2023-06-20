// Code generated by counterfeiter. DO NOT EDIT.
package libfakes

import (
	"sync"

	"github.com/cnoe-io/cnoe-cli/pkg/lib"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type FakeIK8sClient struct {
	CRDsStub        func(string, string, string) (*unstructured.UnstructuredList, error)
	cRDsMutex       sync.RWMutex
	cRDsArgsForCall []struct {
		arg1 string
		arg2 string
		arg3 string
	}
	cRDsReturns struct {
		result1 *unstructured.UnstructuredList
		result2 error
	}
	cRDsReturnsOnCall map[int]struct {
		result1 *unstructured.UnstructuredList
		result2 error
	}
	PodsStub        func(string) (*v1.PodList, error)
	podsMutex       sync.RWMutex
	podsArgsForCall []struct {
		arg1 string
	}
	podsReturns struct {
		result1 *v1.PodList
		result2 error
	}
	podsReturnsOnCall map[int]struct {
		result1 *v1.PodList
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeIK8sClient) CRDs(arg1 string, arg2 string, arg3 string) (*unstructured.UnstructuredList, error) {
	fake.cRDsMutex.Lock()
	ret, specificReturn := fake.cRDsReturnsOnCall[len(fake.cRDsArgsForCall)]
	fake.cRDsArgsForCall = append(fake.cRDsArgsForCall, struct {
		arg1 string
		arg2 string
		arg3 string
	}{arg1, arg2, arg3})
	stub := fake.CRDsStub
	fakeReturns := fake.cRDsReturns
	fake.recordInvocation("CRDs", []interface{}{arg1, arg2, arg3})
	fake.cRDsMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeIK8sClient) CRDsCallCount() int {
	fake.cRDsMutex.RLock()
	defer fake.cRDsMutex.RUnlock()
	return len(fake.cRDsArgsForCall)
}

func (fake *FakeIK8sClient) CRDsCalls(stub func(string, string, string) (*unstructured.UnstructuredList, error)) {
	fake.cRDsMutex.Lock()
	defer fake.cRDsMutex.Unlock()
	fake.CRDsStub = stub
}

func (fake *FakeIK8sClient) CRDsArgsForCall(i int) (string, string, string) {
	fake.cRDsMutex.RLock()
	defer fake.cRDsMutex.RUnlock()
	argsForCall := fake.cRDsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeIK8sClient) CRDsReturns(result1 *unstructured.UnstructuredList, result2 error) {
	fake.cRDsMutex.Lock()
	defer fake.cRDsMutex.Unlock()
	fake.CRDsStub = nil
	fake.cRDsReturns = struct {
		result1 *unstructured.UnstructuredList
		result2 error
	}{result1, result2}
}

func (fake *FakeIK8sClient) CRDsReturnsOnCall(i int, result1 *unstructured.UnstructuredList, result2 error) {
	fake.cRDsMutex.Lock()
	defer fake.cRDsMutex.Unlock()
	fake.CRDsStub = nil
	if fake.cRDsReturnsOnCall == nil {
		fake.cRDsReturnsOnCall = make(map[int]struct {
			result1 *unstructured.UnstructuredList
			result2 error
		})
	}
	fake.cRDsReturnsOnCall[i] = struct {
		result1 *unstructured.UnstructuredList
		result2 error
	}{result1, result2}
}

func (fake *FakeIK8sClient) Pods(arg1 string) (*v1.PodList, error) {
	fake.podsMutex.Lock()
	ret, specificReturn := fake.podsReturnsOnCall[len(fake.podsArgsForCall)]
	fake.podsArgsForCall = append(fake.podsArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.PodsStub
	fakeReturns := fake.podsReturns
	fake.recordInvocation("Pods", []interface{}{arg1})
	fake.podsMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeIK8sClient) PodsCallCount() int {
	fake.podsMutex.RLock()
	defer fake.podsMutex.RUnlock()
	return len(fake.podsArgsForCall)
}

func (fake *FakeIK8sClient) PodsCalls(stub func(string) (*v1.PodList, error)) {
	fake.podsMutex.Lock()
	defer fake.podsMutex.Unlock()
	fake.PodsStub = stub
}

func (fake *FakeIK8sClient) PodsArgsForCall(i int) string {
	fake.podsMutex.RLock()
	defer fake.podsMutex.RUnlock()
	argsForCall := fake.podsArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeIK8sClient) PodsReturns(result1 *v1.PodList, result2 error) {
	fake.podsMutex.Lock()
	defer fake.podsMutex.Unlock()
	fake.PodsStub = nil
	fake.podsReturns = struct {
		result1 *v1.PodList
		result2 error
	}{result1, result2}
}

func (fake *FakeIK8sClient) PodsReturnsOnCall(i int, result1 *v1.PodList, result2 error) {
	fake.podsMutex.Lock()
	defer fake.podsMutex.Unlock()
	fake.PodsStub = nil
	if fake.podsReturnsOnCall == nil {
		fake.podsReturnsOnCall = make(map[int]struct {
			result1 *v1.PodList
			result2 error
		})
	}
	fake.podsReturnsOnCall[i] = struct {
		result1 *v1.PodList
		result2 error
	}{result1, result2}
}

func (fake *FakeIK8sClient) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.cRDsMutex.RLock()
	defer fake.cRDsMutex.RUnlock()
	fake.podsMutex.RLock()
	defer fake.podsMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeIK8sClient) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ lib.IK8sClient = new(FakeIK8sClient)
