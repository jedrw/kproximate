package proxmox

import (
	"context"
	"regexp"

	"github.com/Telmate/proxmox-api-go/proxmox"
)

type ProxmoxMock struct {
	ClusterStats       []HostInformation
	RunningKpNodes     []VmInformation
	KpNodes            []VmInformation
	KpNode             VmInformation
	KpNodeTemplateRef  proxmox.VmRef
	JoinExecPid        int
	QemuExecJoinStatus QemuExecStatus
}

func (p *ProxmoxMock) GetClusterStats(ctx context.Context) ([]HostInformation, error) {
	return p.ClusterStats, nil
}

func (p *ProxmoxMock) GetRunningKpNodes(ctx context.Context, kpNodeName regexp.Regexp) ([]VmInformation, error) {
	return p.RunningKpNodes, nil
}

func (p *ProxmoxMock) GetAllKpNodes(ctx context.Context, kpNodeName regexp.Regexp) ([]VmInformation, error) {
	return p.KpNodes, nil
}

func (p *ProxmoxMock) GetKpNode(ctx context.Context, name string, kpNodeName regexp.Regexp) (VmInformation, error) {
	return p.KpNode, nil
}

func (p *ProxmoxMock) GetKpNodeTemplateRef(ctx context.Context, kpNodeTemplateName string, LocalTemplateStorage bool, cloneTargetNode string) (*proxmox.VmRef, error) {
	return &p.KpNodeTemplateRef, nil
}

func (p *ProxmoxMock) NewKpNode(ctx context.Context, newKpNodeName string, targetNode string, kpNodeParams map[string]interface{}, usingLocalStorage bool, kpNodeTemplateName string, kpJoinCommand string) error {
	return nil
}

func (p *ProxmoxMock) DeleteKpNode(ctx context.Context, name string, kpNodeName regexp.Regexp) error {
	return nil
}

func (p *ProxmoxMock) QemuExecJoin(ctx context.Context, nodeName string, joinCommand string) (int, error) {
	return p.JoinExecPid, nil
}

func (p *ProxmoxMock) GetQemuExecJoinStatus(ctx context.Context, nodeName string, pid int) (QemuExecStatus, error) {
	return p.QemuExecJoinStatus, nil
}

func (p *ProxmoxMock) CheckNodeReady(ctx context.Context, nodeName string) error {
	return nil
}

func GetNextID(ctx context.Context, currentID *proxmox.GuestID) (nextID proxmox.GuestID, err error) {
	return proxmox.GuestID(1), nil
}
