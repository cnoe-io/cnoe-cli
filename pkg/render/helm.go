package render

import k8sexec "k8s.io/utils/exec"

type Helm struct {
	Id        int64
	Path      string
	Name      string
	Deps      []int64
	Env       map[string]string
	executor  k8sexec.Interface
	ExecPath  string
	manifests []byte
}
